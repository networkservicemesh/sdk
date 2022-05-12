// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	nsclient "github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	tick    = 10 * time.Millisecond
	timeout = 10 * time.Second
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
			// nolint:scopelint
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
	require.Eventually(t, func() bool { return counter.UniqueRequests() == 1 }, timeout, tick)
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
			// nolint:scopelint
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
			// nolint:scopelint
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
		require.Equal(t, 1, counter.Closes())
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

	require.Equal(t, 3, counter.Requests())
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
			// nolint:scopelint
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

	if withNSEExpiration {
		builder = builder.SetRegistryExpiryDuration(time.Second / 2)
	}

	domain := builder.Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseCtx, nseCtxCancel := context.WithCancel(ctx)

	domain.Nodes[0].NewEndpoint(nseCtx, defaultRegistryEndpoint(nsReg.Name), sandbox.GenerateTestToken)

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
		c := client.NewNetworkServiceEndpointRegistryClient(ctx,
			client.WithClientURL(domain.Nodes[0].NSMgr.URL),
			client.WithDialOptions(sandbox.DialOptions(sandbox.WithTokenGenerator(sandbox.GenerateTestToken))...))

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

	for _, fwd := range domain.Nodes[0].Forwarders {
		fwd.Cancel()
	}

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
			// nolint:scopelint
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

	for i := 0; i < 10; i++ {
		request.Connection = conn.Clone()
		conn, err = nsc.Request(ctx, request.Clone())

		require.NoError(t, err)
		require.Equal(t, expectedForwarderName, conn.GetPath().GetPathSegments()[pathSegmentCount-2].Name)

		domain.Nodes[nodeNum].NSMgr.Restart()

		_, err = domain.Nodes[nodeNum].NSMgr.NetworkServiceEndpointRegistryServer().Register(ctx, &registry.NetworkServiceEndpoint{
			Name:                nseReg.Name,
			Url:                 nseEntry.URL.String(),
			NetworkServiceNames: nseReg.NetworkServiceNames,
		})
		require.NoError(t, err)

		_, err = domain.Nodes[nodeNum].NSMgr.NetworkServiceEndpointRegistryServer().Register(ctx, &registry.NetworkServiceEndpoint{
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
