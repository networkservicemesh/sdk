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

package nsmgr_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	nsclient "github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkresponse"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	tick       = 10 * time.Millisecond
	timeout    = 10 * time.Second
	labelKey   = "key"
	labelValue = "value"
)

func TestNSMGR_HealEndpoint(t *testing.T) {
	var samples = []struct {
		name    string
		nodeNum int
	}{
		{
			name:    "Local New",
			nodeNum: 0,
		},
		{
			name:    "Remote New",
			nodeNum: 1,
		},
	}

	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			testNSMGRHealEndpoint(t, sample.nodeNum)
		})
	}
}

func testNSMGRHealEndpoint(t *testing.T, nodeNum int) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(nodeNum + 1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counter := new(count.Server)
	nse := domain.Nodes[nodeNum].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())

	nse.Cancel()

	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	nseReg2.Name += "-2"
	domain.Nodes[nodeNum].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, counter)

	// Wait reconnecting to the new NSE
	require.Eventually(t, checkSecondRequestsReceived(counter.UniqueRequests), timeout, tick)
	require.Equal(t, 2, counter.UniqueRequests())
	closes := counter.UniqueCloses()

	// Check refresh
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// Close with old connection
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	require.Equal(t, 2, counter.UniqueRequests())
	require.Equal(t, closes+1, counter.UniqueCloses())
}

func TestNSMGRHealEndpoint_DataPlaneBroken_CtrlPlaneBroken(t *testing.T) {
	// This the same test as above but here we explicitly provided livenessCheck function
	// The above test is for nil livenessCheck

	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counter := new(count.Server)
	nse := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.Name)

	livenessCheck := func(ctx context.Context, conn *networkservice.Connection) bool {
		return false
	}

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken,
		nsclient.WithHealClient(heal.NewClient(ctx,
			heal.WithLivenessCheck(livenessCheck))))

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())

	nse.Cancel()

	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	nseReg2.Name += "-2"
	domain.Nodes[0].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, counter)

	// Wait reconnecting to the new NSE
	require.Eventually(t, checkSecondRequestsReceived(counter.UniqueRequests), timeout, tick)
	require.Equal(t, 2, counter.UniqueRequests())
	closes := counter.UniqueCloses()

	// Check refresh
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// Close with old connection
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	require.Equal(t, 2, counter.UniqueRequests())
	require.Equal(t, closes+1, counter.UniqueCloses())
}

func TestNSMGRHealEndpoint_DataPlaneBroken_CtrlPlaneHealthy(t *testing.T) {
	// This the same test as above but here we explicitly provided livenessCheck function
	// The above test is for nil livenessCheck

	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counter := new(count.Server)
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.Name)

	livenessCheck := func(ctx context.Context, conn *networkservice.Connection) bool { return false }

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken,
		nsclient.WithHealClient(heal.NewClient(ctx, heal.WithLivenessCheck(livenessCheck))))

	// Connect to the first NSE.
	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.Requests())

	// Create the second NSE.
	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	nseReg2.Name += "-2"
	domain.Nodes[0].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, counter)

	require.Eventually(t, checkSecondRequestsReceived(counter.UniqueRequests), timeout, tick)
	require.Equal(t, 2, counter.UniqueRequests())
	closes := counter.UniqueCloses()

	// Check refresh
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// Close with old connection
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	require.Equal(t, 2, counter.UniqueRequests())
	require.Equal(t, closes+1, counter.UniqueCloses())
}

func TestNSMGRHealEndpoint_DatapathHealthy_CtrlPlaneBroken(t *testing.T) {
	t.Skip("https://github.com/networkservicemesh/sdk/issues/1573")

	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counter := new(count.Server)
	nse := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.Name)

	livenessCheck := func(ctx context.Context, conn *networkservice.Connection) bool { return true }

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken,
		nsclient.WithHealClient(heal.NewClient(ctx,
			heal.WithLivenessCheck(livenessCheck))))

	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())

	nse.Cancel()

	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	nseReg2.Name += "-2"
	domain.Nodes[0].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, counter)

	// Should not connect to new NSE
	require.Never(t, func() bool { return counter.UniqueRequests() > 1 }, time.Second*2, tick)
	require.Equal(t, 1, counter.UniqueRequests())
}

func TestNSMGR_HealForwarder(t *testing.T) {
	var samples = []struct {
		name    string
		nodeNum int
	}{
		{
			name:    "Local New",
			nodeNum: 0,
		},
		{
			name:    "Remote New",
			nodeNum: 1,
		},
	}

	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			testNSMGRHealForwarder(t, sample.nodeNum)
		})
	}
}

func testNSMGRHealForwarder(t *testing.T, nodeNum int) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(2).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	counter := new(count.Server)
	domain.Nodes[1].NewEndpoint(ctx, defaultRegistryEndpoint(nsReg.Name), sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())

	for _, forwarder := range domain.Nodes[nodeNum].Forwarders {
		forwarder.Cancel()
		break
	}

	forwarderReg := &registry.NetworkServiceEndpoint{
		Name:                sandbox.UniqueName("forwarder-2"),
		NetworkServiceNames: []string{"forwarder"},
	}
	domain.Nodes[nodeNum].NewForwarder(ctx, forwarderReg, sandbox.GenerateTestToken)

	// Wait reconnecting through the new Forwarder
	require.Eventually(t, checkSecondRequestsReceived(counter.Requests), timeout, tick)
	require.Equal(t, 2, counter.Requests())
	closes := counter.UniqueCloses()

	// Check refresh
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// Close with old connection
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	require.Equal(t, 3, counter.Requests())
	require.Equal(t, closes+1, counter.Closes())
}

func TestNSMGR_HealNSMgr(t *testing.T) {
	var samples = []struct {
		name     string
		nodeNum  int
		restored bool
	}{
		{
			name:     "Local Restored",
			nodeNum:  0,
			restored: true,
		},
		{
			name:    "Remote New",
			nodeNum: 1,
		},
	}

	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			testNSMGRHealNSMgr(t, sample.nodeNum, sample.restored)
		})
	}
}

func testNSMGRHealNSMgr(t *testing.T, nodeNum int, restored bool) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(3).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counter := new(count.Server)
	domain.Nodes[1].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	if !restored {
		nseReg2 := defaultRegistryEndpoint(nsReg.Name)
		nseReg2.Name += "-2"

		domain.Nodes[2].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, counter)

		domain.Nodes[nodeNum].NSMgr.Cancel()
	} else {
		domain.Nodes[nodeNum].NSMgr.Restart()
	}

	var closes int
	if restored {
		// Wait reconnecting through the restored NSMgr
		require.Eventually(t, checkSecondRequestsReceived(counter.Requests), timeout, tick)
		require.Equal(t, 2, counter.Requests())
	} else {
		// Wait reconnecting through the new NSMgr
		require.Eventually(t, checkSecondRequestsReceived(counter.UniqueRequests), timeout, tick)
		require.Equal(t, 2, counter.UniqueRequests())
		closes = counter.UniqueCloses()
	}

	// Check refresh
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// Close with old connection
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	if restored {
		require.Equal(t, 3, counter.Requests())
		require.Equal(t, 2, counter.Closes())
	} else {
		require.Equal(t, 2, counter.UniqueRequests())
		require.Equal(t, closes+1, counter.UniqueCloses())
	}
}

func TestNSMGR_HealRegistry(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counter := new(count.Server)
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// 1. Restart Registry
	domain.Registry.Restart()

	// 2. Check refresh
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// 3. Check new client request
	nsc = domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return counter.Requests() >= 3
	}, timeout, tick)
}

func TestNSMGR_CloseHeal(t *testing.T) {
	var samples = []struct {
		name              string
		withNSEExpiration bool
	}{
		{
			name: "Without NSE expiration",
		},
		{
			name:              "With NSE expiration",
			withNSEExpiration: true,
		},
	}

	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			testNSMGRCloseHeal(t, sample.withNSEExpiration)
		})
	}
}

func testNSMGRCloseHeal(t *testing.T, withNSEExpiration bool) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	builder := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil)

	domain := builder.Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseCtx, nseCtxCancel := context.WithTimeout(ctx, time.Second/2)
	if withNSEExpiration {
		// NSE will be unregistered after (tokenTimeout - registerTimeout)
		domain.Nodes[0].NewEndpoint(nseCtx, defaultRegistryEndpoint(nsReg.Name), sandbox.GenerateExpiringToken(time.Second))
	} else {
		domain.Nodes[0].NewEndpoint(nseCtx, defaultRegistryEndpoint(nsReg.Name), sandbox.GenerateTestToken)
	}

	request := defaultRequest(nsReg.Name)

	nscCtx, nscCtxCancel := context.WithCancel(ctx)

	nsc := domain.Nodes[0].NewClient(nscCtx, sandbox.GenerateTestToken)

	reqCtx, reqCancel := context.WithTimeout(ctx, time.Second)
	defer reqCancel()

	// 1. Request
	conn, err := nsc.Request(reqCtx, request.Clone())
	require.NoError(t, err)

	ignoreCurrent := goleak.IgnoreCurrent()

	// 2. Refresh
	request.Connection = conn

	conn, err = nsc.Request(reqCtx, request.Clone())
	require.NoError(t, err)

	// 3. Stop endpoint and wait for the heal to start
	nseCtxCancel()
	time.Sleep(100 * time.Millisecond)

	if withNSEExpiration {
		// 3.1 Wait for the endpoint expiration
		time.Sleep(time.Second)
		c := registryclient.NewNetworkServiceEndpointRegistryClient(ctx,
			registryclient.WithClientURL(domain.Nodes[0].NSMgr.URL),
			registryclient.WithDialOptions(sandbox.DialOptions(sandbox.WithTokenGenerator(sandbox.GenerateTestToken))...))

		stream, err := c.Find(ctx, &registry.NetworkServiceEndpointQuery{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name: "final-endpoint",
			},
		})

		require.NoError(t, err)

		require.Len(t, registry.ReadNetworkServiceEndpointList(stream), 0)
	}

	// 4. Close connection
	_, _ = nsc.Close(nscCtx, conn.Clone())

	nscCtxCancel()

	require.Eventually(t, func() bool {
		logrus.Error(goleak.Find())
		return goleak.Find(ignoreCurrent) == nil
	}, timeout, tick)

	require.NoError(t, ctx.Err())
}

func checkSecondRequestsReceived(requestsDone func() int) func() bool {
	return func() bool {
		return requestsDone() >= 2
	}
}

func Test_ForwarderShouldBeSelectedCorrectlyOnNSMgrRestart(t *testing.T) {
	var samples = []struct {
		name             string
		nodeNum          int
		pathSegmentCount int
	}{
		{
			name:             "Local",
			nodeNum:          0,
			pathSegmentCount: 4,
		},
		{
			name:             "Remote",
			nodeNum:          1,
			pathSegmentCount: 6,
		},
	}

	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			testForwarderShouldBeSelectedCorrectlyOnNSMgrRestart(t, sample.nodeNum, sample.pathSegmentCount)
		})
	}
}

func testForwarderShouldBeSelectedCorrectlyOnNSMgrRestart(t *testing.T, nodeNum, pathSegmentCount int) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(nodeNum + 1).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		Build()

	var expectedForwarderName string

	require.Len(t, domain.Nodes[nodeNum].Forwarders, 1)
	for k := range domain.Nodes[nodeNum].Forwarders {
		expectedForwarderName = k
	}

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	_, err := nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-ns",
	})
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "my-nse-1",
		NetworkServiceNames: []string{"my-ns"},
	}

	nseEntry := domain.Nodes[nodeNum].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, nsclient.WithHealClient(null.NewClient()))

	request := defaultRequest("my-ns")

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, pathSegmentCount, len(conn.Path.PathSegments))
	require.Equal(t, expectedForwarderName, conn.GetPath().GetPathSegments()[pathSegmentCount-2].Name)

	nseRegistryClient := registryclient.NewNetworkServiceEndpointRegistryClient(ctx,
		registryclient.WithClientURL(sandbox.CloneURL(domain.Nodes[nodeNum].NSMgr.URL)),
		registryclient.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())))

	for i := 0; i < 10; i++ {
		request.Connection = conn.Clone()
		conn, err = nsc.Request(ctx, request.Clone())

		require.NoError(t, err)
		require.Equal(t, expectedForwarderName, conn.GetPath().GetPathSegments()[pathSegmentCount-2].Name)

		domain.Nodes[nodeNum].NSMgr.Restart()

		_, err = nseRegistryClient.Register(ctx, &registry.NetworkServiceEndpoint{
			Name:                nseReg.Name,
			Url:                 nseEntry.URL.String(),
			NetworkServiceNames: nseReg.NetworkServiceNames,
		})
		require.NoError(t, err)

		_, err = nseRegistryClient.Register(ctx, &registry.NetworkServiceEndpoint{
			Name:                expectedForwarderName,
			Url:                 domain.Nodes[nodeNum].Forwarders[expectedForwarderName].URL.String(),
			NetworkServiceNames: []string{"forwarder"},
			NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
				"forwarder": {
					Labels: map[string]string{
						"p2p": "true",
					},
				},
			},
		})
		require.NoError(t, err)

		domain.Nodes[nodeNum].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
			Name:                sandbox.UniqueName(fmt.Sprintf("%v-forwarder", i)),
			NetworkServiceNames: []string{"forwarder"},
			NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
				"forwarder": {
					Labels: map[string]string{
						"p2p": "true",
					},
				},
			},
		}, sandbox.GenerateTestToken)
	}
}

func TestNSMGR_RefreshFailed_DataPlaneBroken(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := defaultRegistryService(t.Name())
	nsReg, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counter1 := new(count.Server)
	// allow only one successful request
	inject := injecterror.NewServer(injecterror.WithCloseErrorTimes(), injecterror.WithRequestErrorTimes(1, -1))
	isDataplaneHealthy := atomic.Bool{}
	isDataplaneHealthy.Store(true)
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter1, inject)

	request := defaultRequest(nsReg.Name)

	tokenDuration := time.Minute * 15
	clk := clockmock.New(ctx)
	clk.Set(time.Now())

	nsc := domain.Nodes[0].NewClient(ctx,
		sandbox.GenerateExpiringToken(tokenDuration),
		nsclient.WithHealClient(heal.NewClient(ctx,
			heal.WithLivenessCheck(func(ctx context.Context, conn *networkservice.Connection) bool {
				return isDataplaneHealthy.Load()
			}),
			heal.WithLivenessCheckInterval(time.Millisecond*10),
		)),
	)

	requestCtx, requestCalcel := context.WithTimeout(ctx, time.Second)
	requestCtx = clock.WithClock(requestCtx, clk)
	defer requestCalcel()
	conn, err := nsc.Request(requestCtx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter1.Requests())

	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	nseReg2.Name += "-2"

	counter2 := new(count.Server)
	// automatically restore data plane flag to prevent repeated heal
	dataPlaneNotifier := checkresponse.NewServer(t, func(t *testing.T, c *networkservice.Connection) {
		if c != nil {
			isDataplaneHealthy.Store(true)
		}
	})
	domain.Nodes[0].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, dataPlaneNotifier, counter2)

	// refresh interval in this test is expected to be 3 minutes and a few milliseconds
	clk.Add(time.Second * 190)

	// wait till refresh reached NSE, to make sure that initial heal monitor is canceled
	require.Eventually(t, checkSecondRequestsReceived(counter1.Requests), timeout, tick)

	isDataplaneHealthy.Store(false)

	require.Eventually(t, func() bool { return counter2.Requests() >= 1 }, timeout, tick)
	require.Equal(t, 1, counter2.UniqueRequests())
	require.Equal(t, 1, counter2.Requests())

	_, err = nsc.Close(ctx, conn.Clone())
	require.NoError(t, err)
}

// This test shows that healing successfully restores the connection if one of the components is killed during the Request
func TestNSMGR_RefreshFailed_ControlPlaneBroken(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := defaultRegistryService(t.Name())
	nsReg, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counter := new(count.Server)
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.Name)

	tokenDuration := time.Minute * 15
	clk := clockmock.New(ctx)
	clk.Set(time.Now())

	// syncCh is used to catch the situation when the forwarder dies during the Request (after the heal chain element)
	syncCh := make(chan struct{}, 1)

	nsc := domain.Nodes[0].NewClient(ctx,
		sandbox.GenerateExpiringToken(tokenDuration),
		nsclient.WithHealClient(heal.NewClient(ctx)),
		nsclient.WithAdditionalFunctionality(
			checkrequest.NewClient(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
				<-syncCh
			}),
		),
	)

	requestCtx, requestCancel := context.WithTimeout(ctx, time.Second)
	defer requestCancel()

	// allow the first Request
	syncCh <- struct{}{}
	conn, err := nsc.Request(requestCtx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.Requests())

	// refresh interval in this test is expected to be 3 minutes and a few milliseconds
	clk.Add(time.Second * 190)
	// start goroutine that will update mock clock every 50 ms. It is needed for retry refresh
	go func() {
		tickerDuration := time.Millisecond * 50
		tickCh := time.Tick(tickerDuration)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tickCh:
				clk.Add(tickerDuration)
			}
		}
	}()

	// kill the forwarder during the refresh (it is stopped by syncCh). Then continue - the refresh will fail.
	for idx := range domain.Nodes[0].Forwarders {
		forwarder := domain.Nodes[0].Forwarders[idx]
		forwarder.Cancel()
		// wait until the forwarder dies
		require.Eventually(t, func() bool {
			return sandbox.CheckURLFree(forwarder.URL)
		}, timeout, tick)
	}
	close(syncCh)

	// create a new forwarder and allow the healing Request
	forwarderReg := &registry.NetworkServiceEndpoint{
		Name:                sandbox.UniqueName("forwarder-2"),
		NetworkServiceNames: []string{"forwarder"},
	}
	domain.Nodes[0].NewForwarder(ctx, forwarderReg, sandbox.GenerateTestToken)

	// wait till Request reached NSE
	require.Eventually(t, func() bool {
		return counter.Requests() == 2
	}, timeout, tick)

	_, err = nsc.Close(ctx, conn.Clone())
	require.NoError(t, err)
	require.Equal(t, 2, counter.Requests())
}

func TestNSMGRHealEndpoint_CustomReselectFunc(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsReg, err := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken).Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)
	nse := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, nsclient.WithHealClient(heal.NewClient(ctx)),
		nsclient.WithReselectFunc(
			func(request *networkservice.NetworkServiceRequest) {
				request.Connection.Labels = make(map[string]string)
				request.Connection.Labels[labelKey] = labelValue
				request.Connection.NetworkServiceEndpointName = ""
			}))

	request := defaultRequest(nsReg.Name)
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	nse.Cancel()

	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	nseReg2.Name += "-2"
	domain.Nodes[0].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken)

	require.Eventually(t, func() bool {
		resp, err := nsc.Request(ctx, request.Clone())
		return err == nil && resp.Labels[labelKey] == labelValue
	}, timeout, tick)
}
