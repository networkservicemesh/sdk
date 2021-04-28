// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package refresh_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

const (
	expireTimeout = 15 * time.Minute
	testWait      = 100 * time.Millisecond
	testTick      = testWait / 100

	sandboxExpireTimeout = 300 * time.Millisecond
	sandboxMinDuration   = 50 * time.Millisecond
	sandboxStepDuration  = 10 * time.Millisecond
	sandboxTotalTimeout  = 800 * time.Millisecond
)

func testTokenFunc(clockTime clock.Clock, timeout time.Duration) token.GeneratorFunc {
	return func(_ credentials.AuthInfo) (token string, expireTime time.Time, err error) {
		return "", clockTime.Now().Add(timeout), err
	}
}

func TestRefreshClient_ValidRefresh(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.NewMock()
	ctx = clock.WithClock(ctx, clockMock)

	cloneClient := &countClient{
		t: t,
	}
	client := chain.NewNetworkServiceClient(
		serialize.NewClient(),
		updatepath.NewClient("refresh"),
		refresh.NewClient(ctx),
		adapters.NewServerToClient(updatetoken.NewServer(testTokenFunc(clockMock, expireTimeout))),
		cloneClient,
	)

	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
		},
	})

	require.NoError(t, err)
	require.Condition(t, cloneClient.validator(1))

	clockMock.Add(expireTimeout)
	require.Eventually(t, cloneClient.validator(2), testWait, testTick)

	lastRequestConn := cloneClient.GetLastRequest().GetConnection()
	require.Equal(t, conn.Id, lastRequestConn.Id)
	require.Equal(t, len(conn.Path.PathSegments), len(lastRequestConn.Path.PathSegments))
	for i := 0; i < len(conn.Path.PathSegments); i++ {
		connSegment := conn.Path.PathSegments[i]
		lastRequestSegment := lastRequestConn.Path.PathSegments[i]
		require.Condition(t, func() (success bool) {
			return connSegment.Expires.AsTime().Before(lastRequestSegment.Expires.AsTime())
		})
	}
}

func TestRefreshClient_StopRefreshAtClose(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.NewMock()
	ctx = clock.WithClock(ctx, clockMock)

	cloneClient := &countClient{
		t: t,
	}
	client := chain.NewNetworkServiceClient(
		serialize.NewClient(),
		updatepath.NewClient("refresh"),
		refresh.NewClient(ctx),
		adapters.NewServerToClient(updatetoken.NewServer(testTokenFunc(clockMock, expireTimeout))),
		cloneClient,
	)

	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
		},
	})
	require.NoError(t, err)
	require.Condition(t, cloneClient.validator(1))

	clockMock.Add(expireTimeout)
	require.Eventually(t, cloneClient.validator(2), testWait, testTick)

	_, err = client.Close(ctx, conn)
	require.NoError(t, err)

	count := atomic.LoadInt32(&cloneClient.count)

	clockMock.Add(2 * expireTimeout)
	require.Never(t, cloneClient.validator(count+1), testWait, testTick)
}

func TestRefreshClient_RestartsRefreshAtAnotherRequest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.NewMock()
	ctx = clock.WithClock(ctx, clockMock)

	cloneClient := &countClient{
		t: t,
	}
	client := chain.NewNetworkServiceClient(
		serialize.NewClient(),
		updatepath.NewClient("refresh"),
		refresh.NewClient(ctx),
		adapters.NewServerToClient(updatetoken.NewServer(testTokenFunc(clockMock, expireTimeout))),
		cloneClient,
	)

	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
		},
	})
	require.NoError(t, err)
	require.Condition(t, cloneClient.validator(1))

	clockMock.Add(expireTimeout)
	require.Eventually(t, cloneClient.validator(2), testWait, testTick)

	_, err = client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: conn.Clone(),
	})
	require.NoError(t, err)

	count := atomic.LoadInt32(&cloneClient.count)

	for i := 0; i < 10; i++ {
		clockMock.Add(expireTimeout / 10)
	}
	require.Eventually(t, cloneClient.validator(count+1), testWait, testTick)
	require.Never(t, cloneClient.validator(count+5), testWait, testTick)
}

type stressTestConfig struct {
	name                     string
	expireTimeout            time.Duration
	minDuration, maxDuration time.Duration
	tickDuration             time.Duration
	iterations               int
}

func TestRefreshClient_CheckRaceConditions(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	conf := &stressTestConfig{
		name:          "RaceConditions",
		expireTimeout: 2 * time.Millisecond,
		minDuration:   0,
		maxDuration:   100 * time.Hour,
		tickDuration:  8100 * time.Microsecond,
		iterations:    100,
	}

	refreshTester := newRefreshTesterServer(t, conf.minDuration, conf.maxDuration)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := next.NewNetworkServiceClient(
		serialize.NewClient(),
		updatepath.NewClient("foo"),
		refresh.NewClient(ctx),
		adapters.NewServerToClient(updatetoken.NewServer(sandbox.GenerateExpiringToken(conf.expireTimeout))),
		adapters.NewServerToClient(refreshTester),
	)

	generateRequests(t, client, refreshTester, conf.iterations, conf.tickDuration)
}

func TestRefreshClient_Sandbox(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), sandboxTotalTimeout)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(2).
		SetContext(ctx).
		SetRegistryProxySupplier(nil).
		SetTokenGenerateFunc(sandbox.GenerateTestToken).
		Build()

	nsReg := &registry.NetworkService{
		Name: "my-service-remote",
	}

	_, err := domain.Nodes[0].NSRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{nsReg.Name},
	}

	refreshSrv := newRefreshTesterServer(t, sandboxMinDuration, sandboxExpireTimeout)
	_, err = domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, refreshSrv)
	require.NoError(t, err)

	nscTokenGenerator := sandbox.GenerateExpiringToken(sandboxExpireTimeout)
	nsc := domain.Nodes[1].NewClient(ctx, nscTokenGenerator)

	refreshSrv.beforeRequest("test-conn")
	_, err = nsc.Request(ctx, mkRequest("test-conn", nil))
	require.NoError(t, err)
	refreshSrv.afterRequest()

	refreshSrv.beforeRequest("test-conn")
	require.Eventually(t, func() bool {
		return refreshSrv.getState() == testRefreshStateDoneRequest
	}, sandboxTotalTimeout, sandboxStepDuration)
}
