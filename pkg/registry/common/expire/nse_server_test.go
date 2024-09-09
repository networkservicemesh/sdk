// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023-2024 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package expire_test

import (
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/expire"
	"github.com/networkservicemesh/sdk/pkg/registry/common/localbypass"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/inject/injectpeertoken"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

const (
	expireTimeout = time.Second
	nseName       = "nse"
	testWait      = 100 * time.Millisecond
	testTick      = testWait / 100
)

func find(ctx context.Context, c registry.NetworkServiceEndpointRegistryClient) (nses []*registry.NetworkServiceEndpoint, err error) {
	stream, err := c.Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	if err != nil {
		return nil, err
	}

	var nseResp *registry.NetworkServiceEndpointResponse
	for nseResp, err = stream.Recv(); err == nil; nseResp, err = stream.Recv() {
		nses = append(nses, nseResp.GetNetworkServiceEndpoint())
	}

	if err != io.EOF {
		return nil, err
	}

	return nses, nil
}

func generateTestToken(ctx context.Context, duration time.Duration) token.GeneratorFunc {
	return func(_ credentials.AuthInfo) (string, time.Time, error) {
		expireTime := clock.FromContext(ctx).Now().Add(duration).Local()

		claims := jwt.RegisteredClaims{
			Subject:   "spiffe://test.com/subject",
			ExpiresAt: jwt.NewNumericDate(expireTime),
		}

		tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("supersecret"))
		return tok, expireTime, err
	}
}

func TestExpireNSEServer_ShouldCorrectlySetExpirationTime_InRemoteCase(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	s := next.NewNetworkServiceEndpointRegistryServer(
		injectpeertoken.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
		updatepath.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
		begin.NewNetworkServiceEndpointRegistryServer(),
		expire.NewNetworkServiceEndpointRegistryServer(ctx),
		new(remoteNSEServer),
	)

	resp, err := s.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: nseName,
	})
	require.NoError(t, err)

	require.Equal(t, expireTimeout, clockMock.Until(resp.GetExpirationTime().AsTime().Local()))
}

func TestExpireNSEServer_ShouldUseLessExpirationTimeFromInput_AndWork(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	mem := memory.NewNetworkServiceEndpointRegistryServer()

	s := next.NewNetworkServiceEndpointRegistryServer(
		injectpeertoken.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
		updatepath.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
		begin.NewNetworkServiceEndpointRegistryServer(),
		expire.NewNetworkServiceEndpointRegistryServer(ctx),
		mem,
	)

	resp, err := s.Register(ctx, &registry.NetworkServiceEndpoint{
		Name:           nseName,
		ExpirationTime: timestamppb.New(clockMock.Now().Add(expireTimeout / 2)),
	})
	require.NoError(t, err)

	require.Equal(t, clockMock.Until(resp.GetExpirationTime().AsTime()), expireTimeout/2)

	clockMock.Add(expireTimeout / 2)
	require.Eventually(t, func() bool {
		nses, err := find(ctx, adapters.NetworkServiceEndpointServerToClient(mem))
		return err == nil && len(nses) == 0
	}, testWait, testTick)
}

func TestExpireNSEServer_ShouldSetDefaultExpiration(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	s := next.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		expire.NewNetworkServiceEndpointRegistryServer(ctx, expire.WithDefaultExpiration(expireTimeout)),
	)

	resp, err := s.Register(ctx, &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)

	require.Equal(t, expireTimeout, clockMock.Until(resp.GetExpirationTime().AsTime()))
}

func TestExpireNSEServer_ShouldUseLessExpirationTime_DefaultExpireTimeout(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	s := next.NewNetworkServiceEndpointRegistryServer(
		injectpeertoken.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
		updatepath.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
		begin.NewNetworkServiceEndpointRegistryServer(),
		expire.NewNetworkServiceEndpointRegistryServer(ctx, expire.WithDefaultExpiration(expireTimeout/2)),
	)

	resp, err := s.Register(ctx, &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)

	require.Equal(t, expireTimeout/2, clockMock.Until(resp.GetExpirationTime().AsTime()))
}

func TestExpireNSEServer_ShouldUseLessExpirationTimeFromResponse(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	s := next.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		injectpeertoken.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
		updatepath.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
		expire.NewNetworkServiceEndpointRegistryServer(ctx),
		new(remoteNSEServer), // <-- GRPC invocation
		begin.NewNetworkServiceEndpointRegistryServer(),
		updatepath.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout/2)),
		expire.NewNetworkServiceEndpointRegistryServer(ctx),
	)

	resp, err := s.Register(ctx, &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)

	require.Equal(t, expireTimeout/2, clockMock.Until(resp.GetExpirationTime().AsTime()))
}

func TestExpireNSEServer_ShouldRemoveNSEAfterExpirationTime(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	mem := memory.NewNetworkServiceEndpointRegistryServer()

	s := next.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		injectpeertoken.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
		updatepath.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
		expire.NewNetworkServiceEndpointRegistryServer(ctx),
		new(remoteNSEServer), // <-- GRPC invocation
		mem,
	)

	_, err := s.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: nseName,
	})
	require.NoError(t, err)

	c := adapters.NetworkServiceEndpointServerToClient(mem)

	nses, err := find(ctx, c)
	require.NoError(t, err)
	require.Len(t, nses, 1)
	require.Equal(t, nseName, nses[0].GetName())

	clockMock.Add(expireTimeout)
	require.Eventually(t, func() bool {
		nses, err = find(ctx, c)
		return err == nil && len(nses) == 0
	}, testWait, testTick)
}

func TestExpireNSEServer_DataRace(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mem := memory.NewNetworkServiceEndpointRegistryServer()

	s := next.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		expire.NewNetworkServiceEndpointRegistryServer(ctx),
		localbypass.NewNetworkServiceEndpointRegistryServer("tcp://0.0.0.0"),
		mem,
	)

	for i := 0; i < 200; i++ {
		_, err := s.Register(context.Background(), &registry.NetworkServiceEndpoint{
			Name: nseName,
			Url:  "tcp://1.1.1.1",
		})
		require.NoError(t, err)
	}

	require.Eventually(t, func() bool {
		nses, err := find(context.Background(), adapters.NetworkServiceEndpointServerToClient(mem))
		return err == nil && len(nses) == 0
	}, testWait, testTick)
}

func TestExpireNSEServer_RefreshFailure(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	c := next.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		refresh.NewNetworkServiceEndpointRegistryClient(ctx),
		adapters.NetworkServiceEndpointServerToClient(next.NewNetworkServiceEndpointRegistryServer(
			new(remoteNSEServer), // <-- GRPC invocation
			injectpeertoken.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
			updatepath.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
			begin.NewNetworkServiceEndpointRegistryServer(),
			expire.NewNetworkServiceEndpointRegistryServer(ctx),
			injecterror.NewNetworkServiceEndpointRegistryServer(
				injecterror.WithRegisterErrorTimes(1, -1),
				injecterror.WithFindErrorTimes(),
				injecterror.WithUnregisterErrorTimes(),
			),
			memory.NewNetworkServiceEndpointRegistryServer(),
		)),
	)

	_, err := c.Register(ctx, &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)

	clockMock.Add(expireTimeout)
	require.Eventually(t, func() bool {
		nses, err := find(ctx, c)
		return err == nil && len(nses) == 0
	}, testWait, testTick)
}

func TestExpireNSEServer_UnregisterFailure(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	mem := memory.NewNetworkServiceEndpointRegistryServer()

	s := next.NewNetworkServiceEndpointRegistryServer(
		injectpeertoken.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
		updatepath.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
		begin.NewNetworkServiceEndpointRegistryServer(),
		expire.NewNetworkServiceEndpointRegistryServer(ctx),
		injecterror.NewNetworkServiceEndpointRegistryServer(
			injecterror.WithRegisterErrorTimes(),
			injecterror.WithFindErrorTimes(),
			injecterror.WithUnregisterErrorTimes(0),
		),
		expire.NewNetworkServiceEndpointRegistryServer(ctx),
		mem,
	)

	nse, err := s.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: nseName,
	})
	require.NoError(t, err)

	_, err = s.Unregister(ctx, nse)
	require.Error(t, err)

	c := adapters.NetworkServiceEndpointServerToClient(mem)

	nses, err := find(ctx, c)
	require.NoError(t, err)
	require.Len(t, nses, 1)
	require.Equal(t, nseName, nses[0].GetName())

	clockMock.Add(expireTimeout)
	require.Eventually(t, func() bool {
		nses, err = find(ctx, c)
		return err == nil && len(nses) == 0
	}, testWait, testTick)
}

func TestExpireNSEServer_RefreshKeepsNoUnregister(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	unregisterServer := new(unregisterNSEServer)

	c := next.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		refresh.NewNetworkServiceEndpointRegistryClient(ctx),
		adapters.NetworkServiceEndpointServerToClient(
			next.NewNetworkServiceEndpointRegistryServer(
				// NSMgr chain
				new(remoteNSEServer), // <-- GRPC invocation
				injectpeertoken.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
				updatepath.NewNetworkServiceEndpointRegistryServer(generateTestToken(ctx, expireTimeout)),
				expire.NewNetworkServiceEndpointRegistryServer(ctx),
				unregisterServer,
			)),
	)

	_, err := c.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: nseName,
	})
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		clockMock.Add(expireTimeout*2/3 + time.Millisecond)
		require.Never(t, func() bool {
			return atomic.LoadInt32(&unregisterServer.unregisterCount) > 0
		}, testWait, testTick)
	}
}

type remoteNSEServer struct {
	registry.NetworkServiceEndpointRegistryServer
}

func (s *remoteNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to ctx.Err")
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse.Clone())
}

func (s *remoteNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	if err := server.Context().Err(); err != nil {
		return errors.Wrap(err, "failed to server.Context().Err()")
	}
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *remoteNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to ctx.Err")
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse.Clone())
}

type unregisterNSEServer struct {
	unregisterCount int32
}

func (s *unregisterNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *unregisterNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *unregisterNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*emptypb.Empty, error) {
	atomic.AddInt32(&s.unregisterCount, 1)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

type key string

const (
	ctxValue     = "ctxValue"
	valueKey key = "ctxKey"
)

type extendCtxNSEServer struct {
	ctxValue string
	counter  atomic.Bool
}

func (s *extendCtxNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	ctx = context.WithValue(ctx, valueKey, ctxValue)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *extendCtxNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *extendCtxNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*emptypb.Empty, error) {
	var ok bool
	s.ctxValue, ok = ctx.Value(valueKey).(string)
	if ok {
		s.counter.Store(true)
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

func TestExpireNSEServer_ExtendedContext(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	extendCtxNSEServer := &extendCtxNSEServer{}
	server := next.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		extendCtxNSEServer,
		expire.NewNetworkServiceEndpointRegistryServer(ctx),
	)
	_, err := server.Register(ctx, &registry.NetworkServiceEndpoint{Name: nseName})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		val := extendCtxNSEServer.counter.Load()
		return val == true
	}, time.Second, time.Millisecond*200)
}
