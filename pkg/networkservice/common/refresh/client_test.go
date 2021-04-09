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

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	expireTimeout     = 300 * time.Millisecond
	eventuallyTimeout = expireTimeout
	tickTimeout       = 10 * time.Millisecond
	neverTimeout      = expireTimeout
	maxDuration       = 100 * time.Hour

	sandboxExpireTimeout = 300 * time.Millisecond
	sandboxMinDuration   = 50 * time.Millisecond
	sandboxStepDuration  = 10 * time.Millisecond
	sandboxTotalTimeout  = 800 * time.Millisecond
)

func TestRefreshClient_ValidRefresh(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cloneClient := &countClient{
		t: t,
	}
	client := chain.NewNetworkServiceClient(
		serialize.NewClient(),
		updatepath.NewClient("refresh"),
		refresh.NewClient(ctx),
		adapters.NewServerToClient(updatetoken.NewServer(sandbox.GenerateExpiringToken(expireTimeout))),
		cloneClient,
	)

	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
		},
	})

	require.NoError(t, err)
	require.Condition(t, cloneClient.validator(1))

	require.Eventually(t, cloneClient.validator(2), eventuallyTimeout, tickTimeout)

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

	cloneClient := &countClient{
		t: t,
	}
	client := chain.NewNetworkServiceClient(
		serialize.NewClient(),
		updatepath.NewClient("refresh"),
		refresh.NewClient(ctx),
		adapters.NewServerToClient(updatetoken.NewServer(sandbox.GenerateExpiringToken(expireTimeout))),
		cloneClient,
	)

	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
		},
	})
	require.NoError(t, err)
	require.Condition(t, cloneClient.validator(1))

	require.Eventually(t, cloneClient.validator(2), eventuallyTimeout, tickTimeout)

	_, err = client.Close(ctx, conn)
	require.NoError(t, err)

	count := atomic.LoadInt32(&cloneClient.count)
	require.Never(t, cloneClient.validator(count+1), neverTimeout, tickTimeout)
}

func TestRefreshClient_RestartsRefreshAtAnotherRequest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cloneClient := &countClient{
		t: t,
	}
	client := chain.NewNetworkServiceClient(
		serialize.NewClient(),
		updatepath.NewClient("refresh"),
		refresh.NewClient(ctx),
		adapters.NewServerToClient(updatetoken.NewServer(sandbox.GenerateExpiringToken(expireTimeout))),
		cloneClient,
	)

	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
		},
	})
	require.NoError(t, err)
	require.Condition(t, cloneClient.validator(1))

	require.Eventually(t, cloneClient.validator(2), eventuallyTimeout, tickTimeout)

	_, err = client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: conn.Clone(),
	})
	require.NoError(t, err)

	count := atomic.LoadInt32(&cloneClient.count)
	require.Eventually(t, cloneClient.validator(count+1), eventuallyTimeout, tickTimeout)
	require.Never(t, cloneClient.validator(count+5), eventuallyTimeout, tickTimeout)
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
		maxDuration:   maxDuration,
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

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		Build()

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{"my-service-remote"},
	}

	refreshSrv := newRefreshTesterServer(t, sandboxMinDuration, sandboxExpireTimeout)
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.DefaultTokenTimeout, refreshSrv)

	nsc := domain.Nodes[1].NewClient(ctx, sandboxExpireTimeout)

	refreshSrv.beforeRequest("test-conn")
	_, err := nsc.Request(ctx, mkRequest("test-conn", nil))
	require.NoError(t, err)
	refreshSrv.afterRequest()

	refreshSrv.beforeRequest("test-conn")
	require.Eventually(t, func() bool {
		return refreshSrv.getState() == testRefreshStateDoneRequest
	}, sandboxTotalTimeout, sandboxStepDuration)
}
