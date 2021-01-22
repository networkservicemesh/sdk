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
	"github.com/networkservicemesh/sdk/pkg/tools/logger"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	expireTimeout     = 500 * time.Millisecond
	eventuallyTimeout = expireTimeout
	tickTimeout       = 50 * time.Millisecond
	neverTimeout      = 5 * expireTimeout
	maxDuration       = 100 * time.Hour

	sandboxExpireTimeout = 10 * time.Second
	sandboxMinDuration   = 1 * time.Second
	sandboxStepDuration  = 10 * time.Second
	sandboxRequests      = 5
	sandboxTotalTimeout  = 80 * time.Second
)

func TestRefreshClient_StopRefreshAtClose(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cloneClient := &countClient{
		t: t,
	}
	client := chain.NewNetworkServiceClient(
		serialize.NewClient(),
		updatepath.NewClient("refresh"),
		refresh.NewClient(ctx),
		updatetoken.NewClient(sandbox.GenerateExpiringToken(expireTimeout)),
		cloneClient,
	)

	ctx = logger.WithLog(ctx)
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
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cloneClient := &countClient{
		t: t,
	}
	client := chain.NewNetworkServiceClient(
		serialize.NewClient(),
		updatepath.NewClient("refresh"),
		refresh.NewClient(ctx),
		updatetoken.NewClient(sandbox.GenerateExpiringToken(expireTimeout)),
		cloneClient,
	)

	ctx = logger.WithLog(ctx)
	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
		},
	})
	require.NoError(t, err)
	require.Condition(t, cloneClient.validator(1))

	require.Eventually(t, cloneClient.validator(2), eventuallyTimeout, tickTimeout)

	_, err = client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: conn,
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

func TestRefreshClient_Stress(t *testing.T) {
	table := []stressTestConfig{
		{
			name:          "RaceConditions",
			expireTimeout: 2 * time.Millisecond,
			minDuration:   0,
			maxDuration:   maxDuration,
			tickDuration:  8100 * time.Microsecond,
			iterations:    100,
		},
		{
			name:          "Durations",
			expireTimeout: 500 * time.Millisecond,
			minDuration:   100 * time.Millisecond,
			maxDuration:   500 * time.Millisecond,
			tickDuration:  409 * time.Millisecond,
			iterations:    10,
		},
	}
	for i := range table {
		it := &table[i]
		t.Run(it.name, func(t *testing.T) { runStressTest(t, it) })
	}
}

func runStressTest(t *testing.T, conf *stressTestConfig) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	refreshTester := newRefreshTesterServer(t, conf.minDuration, conf.maxDuration)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := next.NewNetworkServiceClient(
		serialize.NewClient(),
		updatepath.NewClient("foo"),
		refresh.NewClient(ctx),
		updatetoken.NewClient(sandbox.GenerateExpiringToken(conf.expireTimeout)),
		adapters.NewServerToClient(refreshTester),
	)

	generateRequests(t, client, refreshTester, conf.iterations, conf.tickDuration)
}

func TestRefreshClient_Sandbox(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithTimeout(context.Background(), sandboxTotalTimeout)
	defer cancel()

	tokenGenerator := sandbox.GenerateExpiringToken(sandboxExpireTimeout)

	domain := sandbox.NewBuilder(t).
		SetNodesCount(2).
		SetContext(ctx).
		SetRegistryProxySupplier(nil).
		SetTokenGenerateFunc(tokenGenerator).
		Build()
	defer domain.Cleanup()

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{"my-service-remote"},
	}

	refreshSrv := newRefreshTesterServer(t, sandboxMinDuration, sandboxExpireTimeout)
	_, err := sandbox.NewEndpoint(ctx, nseReg, tokenGenerator, domain.Nodes[0].NSMgr, refreshSrv)
	require.NoError(t, err)

	nsc := sandbox.NewClient(ctx, tokenGenerator, domain.Nodes[1].NSMgr.URL)

	generateRequests(t, nsc, refreshSrv, sandboxRequests, sandboxStepDuration)
}
