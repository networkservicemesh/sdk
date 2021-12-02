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
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	countutil "github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injectclock"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
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

func testTokenFunc(clockTime clock.Clock) token.GeneratorFunc {
	return func(_ credentials.AuthInfo) (token string, expireTime time.Time, err error) {
		return "", clockTime.Now().Add(expireTimeout), err
	}
}

func testTokenFuncWithTimeout(clockTime clock.Clock, timeout time.Duration) token.GeneratorFunc {
	return func(_ credentials.AuthInfo) (token string, expireTime time.Time, err error) {
		return "", clockTime.Now().Add(timeout), err
	}
}

type captureTickerDuration struct {
	*clockmock.Mock

	tickerDuration time.Duration
}

func (m *captureTickerDuration) Ticker(d time.Duration) clock.Ticker {
	m.tickerDuration = d
	return m.Mock.Ticker(d)
}

func (m *captureTickerDuration) Reset(t time.Time) {
	m.tickerDuration = 0
	m.Set(t)
}

func testClient(
	ctx context.Context,
	tokenGenerator token.GeneratorFunc,
	clk clock.Clock,
	additionalFunctionality ...networkservice.NetworkServiceClient,
) networkservice.NetworkServiceClient {
	return next.NewNetworkServiceClient(
		append([]networkservice.NetworkServiceClient{
			updatepath.NewClient("refresh"),
			begin.NewClient(),
			metadata.NewClient(),
			injectclock.NewClient(clk),
			refresh.NewClient(ctx),
			adapters.NewServerToClient(
				updatetoken.NewServer(tokenGenerator),
			),
		}, additionalFunctionality...,
		)...,
	)
}

func TestRefreshClient_ValidRefresh(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)

	cloneClient := &countClient{
		t: t,
	}
	client := testClient(ctx, testTokenFunc(clockMock), clockMock, cloneClient)

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

	clockMock := clockmock.New(ctx)

	cloneClient := &countClient{
		t: t,
	}
	client := chain.NewNetworkServiceClient(
		begin.NewClient(),
		metadata.NewClient(),
		injectclock.NewClient(clockMock),
		updatepath.NewClient("refresh"),
		refresh.NewClient(ctx),
		adapters.NewServerToClient(updatetoken.NewServer(testTokenFunc(clockMock))),
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

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	cloneClient := &countClient{
		t: t,
	}
	client := testClient(ctx, testTokenFunc(clockMock), clockMock, cloneClient)

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

	client := testClient(ctx, sandbox.GenerateExpiringToken(conf.expireTimeout), clock.FromContext(ctx), adapters.NewServerToClient(refreshTester))

	generateRequests(t, client, refreshTester, conf.iterations, conf.tickDuration)
}

func TestRefreshClient_Sandbox(t *testing.T) {
	t.Skip("https://github.com/networkservicemesh/sdk/issues/839")

	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), sandboxTotalTimeout)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		SetTokenGenerateFunc(sandbox.GenerateTestToken).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := &registry.NetworkService{
		Name: "my-service-remote",
	}

	_, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{nsReg.Name},
	}

	refreshSrv := newRefreshTesterServer(t, sandboxMinDuration, sandboxExpireTimeout)
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, refreshSrv)

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

func TestRefreshClient_NoRefreshOnFailure(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)

	cloneClient := &countClient{
		t: t,
	}
	client := testClient(ctx, testTokenFunc(clockMock),
		clockMock,
		cloneClient,
		injecterror.NewClient(),
	)

	_, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: new(networkservice.Connection),
	})
	require.Error(t, err)

	clockMock.Add(expireTimeout)

	require.Never(t, cloneClient.validator(2), testWait, testTick)
}

func TestRefreshClient_CalculatesShortestTokenTimeout(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testData := []struct {
		Chain                     []time.Duration
		ExpectedRefreshTimeoutMax time.Duration
		ExpectedRefreshTimeoutMin time.Duration
	}{
		{
			Chain:                     []time.Duration{time.Hour},
			ExpectedRefreshTimeoutMax: 20*time.Minute + time.Second,
			ExpectedRefreshTimeoutMin: 20*time.Minute - time.Second,
		},
		{
			Chain:                     []time.Duration{time.Hour, 3 * time.Minute},
			ExpectedRefreshTimeoutMax: 54*time.Second + time.Second/2.,
			ExpectedRefreshTimeoutMin: 54*time.Second - time.Second/2.,
		},
		{
			Chain:                     []time.Duration{time.Hour, 5 * time.Second, 3 * time.Minute},
			ExpectedRefreshTimeoutMax: 5*time.Second/3. + time.Second/3.,
			ExpectedRefreshTimeoutMin: 5*time.Second/3. - time.Second/3.,
		},
		{
			Chain:                     []time.Duration{200 * time.Millisecond, 1 * time.Minute, 100 * time.Millisecond, time.Hour},
			ExpectedRefreshTimeoutMax: 100*time.Millisecond/3. + 30*time.Millisecond,
			ExpectedRefreshTimeoutMin: 100*time.Millisecond/3. - 30*time.Millisecond,
		},
	}

	timeNow, err := time.Parse("2006-01-02 15:04:05", "2009-11-10 23:00:00")
	require.NoError(t, err)

	clockMock := captureTickerDuration{
		Mock: clockmock.New(ctx),
	}

	var countClient = &countutil.Client{}

	for _, testDataElement := range testData {
		clockMock.Reset(timeNow)

		var pathChain []networkservice.NetworkServiceClient
		var clientChain = []networkservice.NetworkServiceClient{
			begin.NewClient(),
			metadata.NewClient(),
			injectclock.NewClient(&clockMock),
			refresh.NewClient(ctx),
		}

		for _, expireTimeValue := range testDataElement.Chain {
			pathChain = append(pathChain,
				updatepath.NewClient(fmt.Sprintf("test-%v", expireTimeValue)),
				adapters.NewServerToClient(updatetoken.NewServer(testTokenFuncWithTimeout(&clockMock, expireTimeValue))),
			)
		}

		clientChain = append(pathChain, clientChain...)
		clientChain = append(clientChain, countClient)
		client := chain.NewNetworkServiceClient(clientChain...)

		_, err = client.Request(ctx, &networkservice.NetworkServiceRequest{
			Connection: new(networkservice.Connection),
		})
		require.NoError(t, err)

		require.Less(t, clockMock.tickerDuration, testDataElement.ExpectedRefreshTimeoutMax)
		require.Greater(t, clockMock.tickerDuration, testDataElement.ExpectedRefreshTimeoutMin)
	}

	require.Equal(t, countClient.Requests(), len(testData))
}

func TestRefreshClient_RefreshOnRefreshFailure(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)

	cloneClient := &countClient{
		t: t,
	}
	client := testClient(ctx, testTokenFunc(clockMock),
		clockMock,
		cloneClient,
		injecterror.NewClient(injecterror.WithRequestErrorTimes(1, -1)),
	)

	_, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: new(networkservice.Connection),
	})
	require.NoError(t, err)

	clockMock.Add(expireTimeout)

	require.Eventually(t, cloneClient.validator(2), testWait, testTick)

	clockMock.Add(expireTimeout)

	require.Eventually(t, cloneClient.validator(3), testWait, testTick)
}
