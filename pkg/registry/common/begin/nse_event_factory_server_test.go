// Copyright (c) 2022 Cisco and/or its affiliates.
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

package begin_test

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
)

// This test reproduces the situation when refresh changes the eventFactory context.
func TestContextValues_Server(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	checkCtxServ := &checkContextServer{t: t}
	eventFactoryServ := &eventFactoryServer{}
	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		checkCtxServ,
		eventFactoryServ,
		&failedNSEServer{},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set any value to context
	ctx = context.WithValue(ctx, contextKey{}, "value_1")
	checkCtxServ.setExpectedValue("value_1")

	// Do Register with this context
	nse := &registry.NetworkServiceEndpoint{
		Name: "1",
	}
	nse, err := server.Register(ctx, nse.Clone())
	require.NotNil(t, t, nse)
	require.NoError(t, err)

	// Change context value before refresh
	ctx = context.WithValue(ctx, contextKey{}, "value_2")

	// Call refresh that will fail
	nse.Url = failedNSEURLServer
	checkCtxServ.setExpectedValue("value_2")
	_, err = server.Register(ctx, nse.Clone())
	require.Error(t, err)

	// Call refresh from eventFactory. We are expecting the previous value in the context
	checkCtxServ.setExpectedValue("value_1")
	eventFactoryServ.callRefresh()

	// Call refresh that will successful
	nse.Url = ""
	checkCtxServ.setExpectedValue("value_2")
	nse, err = server.Register(ctx, nse.Clone())
	require.NotNil(t, t, nse)
	require.NoError(t, err)

	// Call refresh from eventFactory. We are expecting updated value in the context
	eventFactoryServ.callRefresh()
}

// This test reproduces the situation when Unregister and Register were called at the same time.
func TestRefreshDuringUnregister_Server(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	checkCtxServ := &checkContextServer{t: t}
	eventFactoryServ := &eventFactoryServer{}
	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		checkCtxServ,
		eventFactoryServ,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set any value to context
	ctx = context.WithValue(ctx, contextKey{}, "value_1")
	checkCtxServ.setExpectedValue("value_1")

	// Do Register with this context
	nse := &registry.NetworkServiceEndpoint{
		Name: "1",
	}
	nse, err := server.Register(ctx, nse.Clone())
	require.NotNil(t, t, nse)
	require.NoError(t, err)

	// Change context value before refresh
	ctx = context.WithValue(ctx, contextKey{}, "value_2")
	checkCtxServ.setExpectedValue("value_2")

	// Call Unregister from eventFactory
	eventFactoryServ.callUnregister()

	// Call refresh (should be called at the same time as Unregister)
	nse, err = server.Register(ctx, nse.Clone())
	require.NotNil(t, t, nse)
	require.NoError(t, err)

	// Call refresh from eventFactory. We are expecting updated value in the context
	eventFactoryServ.callRefresh()
}

// This test checks if the timeout for the Register/Unregister called from the event factory is correct.
func TestContextTimeout_Server(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add clockMock to the context
	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	ctx, cancel = context.WithDeadline(ctx, clockMock.Now().Add(time.Second*3))
	defer cancel()

	eventFactoryServ := &eventFactoryServer{}
	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		eventFactoryServ,
		&delayedNSEServer{t: t, clock: clockMock},
	)

	// Do Register
	nse := &registry.NetworkServiceEndpoint{
		Name: "1",
	}
	nse, err := server.Register(ctx, nse.Clone())
	require.NotNil(t, t, nse)
	require.NoError(t, err)

	// Check eventFactory Refresh. We are expecting the same timeout as for register
	eventFactoryServ.callRefresh()

	// Check eventFactory Unregister. We are expecting the same timeout as for register
	eventFactoryServ.callUnregister()
}

type eventFactoryServer struct {
	registry.NetworkServiceEndpointRegistryServer
	ctx context.Context
}

func (e *eventFactoryServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	e.ctx = ctx
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (e *eventFactoryServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*emptypb.Empty, error) {
	// Wait to be sure that reregister was called
	time.Sleep(time.Millisecond * 100)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}

func (e *eventFactoryServer) callUnregister() {
	eventFactory := begin.FromContext(e.ctx)
	eventFactory.Unregister()
}

func (e *eventFactoryServer) callRefresh() {
	eventFactory := begin.FromContext(e.ctx)
	<-eventFactory.Register()
}

type checkContextServer struct {
	registry.NetworkServiceEndpointRegistryServer
	t             *testing.T
	expectedValue string
}

func (c *checkContextServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	require.Equal(c.t, c.expectedValue, ctx.Value(contextKey{}))
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (c *checkContextServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*emptypb.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}

func (c *checkContextServer) setExpectedValue(value string) {
	c.expectedValue = value
}

const failedNSEURLServer = "failedNSE"

type failedNSEServer struct {
	registry.NetworkServiceEndpointRegistryServer
}

func (f *failedNSEServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if in.GetUrl() == failedNSEURLServer {
		return nil, errors.New("failed")
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (f *failedNSEServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*emptypb.Empty, error) {
	if in.GetUrl() == failedNSEURLServer {
		return nil, errors.New("failed")
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}

type delayedNSEServer struct {
	registry.NetworkServiceEndpointRegistryServer
	t              *testing.T
	clock          *clockmock.Mock
	initialTimeout time.Duration
}

func (d *delayedNSEServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	deadline, _ := ctx.Deadline()
	clockTime := clock.FromContext(ctx)
	timeout := clockTime.Until(deadline)

	// Check that context timeout is greater than 0
	require.Greater(d.t, timeout, time.Duration(0))

	// For the first request
	if d.initialTimeout == 0 {
		d.initialTimeout = timeout
	}
	// All requests timeout must be equal the first
	require.Equal(d.t, d.initialTimeout, timeout)

	// Add delay
	d.clock.Add(timeout / 2)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (d *delayedNSEServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*emptypb.Empty, error) {
	require.Greater(d.t, d.initialTimeout, time.Duration(0))

	deadline, _ := ctx.Deadline()
	clockTime := clock.FromContext(ctx)

	require.Equal(d.t, d.initialTimeout, clockTime.Until(deadline))
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}
