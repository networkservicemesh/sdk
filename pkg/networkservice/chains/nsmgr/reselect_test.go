// Copyright (c) 2023 Cisco and/or its affiliates.
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
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

// even if local NSMgr has restarted,
// we expect that all apps after it should get a Close call
// and treat new request as reselect request
func TestReselect_LocalNsmgrRestart(t *testing.T) {
	var samples = []struct {
		name    string
		nodeNum int
	}{
		{
			name:    "Local",
			nodeNum: 1,
		},
		{
			name:    "Remote",
			nodeNum: 2,
		},
	}

	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			// nolint:scopelint
			testReselectWithLocalNsmgrRestart(t, sample.nodeNum)
		})
	}
}

func testReselectWithLocalNsmgrRestart(t *testing.T, nodeNum int) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// in this test we add counters to apps in chain
	// to make sure that in each app Close call goes through the whole chain,
	// without stopping on an error mid-chain
	var counterFwd []*count.Server
	for i := 0; i < nodeNum; i++ {
		counterFwd = append(counterFwd, new(count.Server))
	}

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(nodeNum).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, i int) {
			node.NewNSMgr(ctx, "nsmgr", nil, sandbox.GenerateTestToken, nsmgr.NewServer)
			node.NewForwarder(ctx, &registry.NetworkServiceEndpoint{
				Name:                sandbox.UniqueName("forwarder"),
				NetworkServiceNames: []string{"forwarder"},
			}, sandbox.GenerateTestToken, counterFwd[i])
		}).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counterNse := new(count.Server)
	nse := domain.Nodes[nodeNum-1].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counterNse)

	request := defaultRequest(nsReg.Name)

	counterClient := new(count.Client)
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithAdditionalFunctionality(counterClient))

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	domain.Nodes[0].NSMgr.Restart()

	nse.Cancel()

	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	nseReg2.Name += "-2"
	domain.Nodes[nodeNum-1].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, counterNse)

	// Wait for heal to finish successfully
	require.Eventually(t, checkSecondRequestsReceived(counterNse.UniqueRequests), timeout, tick)
	// Client should try to close connection before reselect
	require.Equal(t, 1, counterClient.UniqueCloses())
	// Forwarder should get a Close, even though NSMgr restarted and didn't pass the Close
	for i := 0; i < nodeNum; i++ {
		require.Equal(t, 1, counterFwd[i].Closes())
	}
	// Old NSE died, new NSE should not get a Close call
	require.Equal(t, 0, counterNse.Closes())

	// Refresh shouldn't cause Close calls
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 0, counterNse.Closes())
	for i := 0; i < nodeNum; i++ {
		require.Equal(t, 1, counterFwd[i].Closes())
	}

	clientCloses := counterClient.Closes()
	// Close should still be able to pass though the whole connection path
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, clientCloses+1, counterClient.Closes())
	require.Equal(t, 1, counterNse.Closes())
	for i := 0; i < nodeNum; i++ {
		require.Equal(t, 1, counterFwd[i].UniqueCloses(), i)
		require.Equal(t, 2, counterFwd[i].Closes(), i)
	}
}
func TestReselect_LocalForwarderRestart(t *testing.T) {
	var samples = []struct {
		name    string
		nodeNum int
	}{
		{
			name:    "Local",
			nodeNum: 1,
		},
		{
			name:    "Remote",
			nodeNum: 2,
		},
	}

	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			// nolint:scopelint
			testReselectWithLocalForwarderRestart(t, sample.nodeNum)
		})
	}
}
func testReselectWithLocalForwarderRestart(t *testing.T, nodeNum int) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// in this test we add counters to apps in chain
	// to make sure that in each app Close call goes through the whole chain,
	// without stopping on an error mid-chain
	var counterFwd []*count.Server
	for i := 0; i < nodeNum; i++ {
		counterFwd = append(counterFwd, new(count.Server))
	}

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(nodeNum).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, i int) {
			node.NewNSMgr(ctx, "nsmgr", nil, sandbox.GenerateTestToken, nsmgr.NewServer)
			node.NewForwarder(ctx, &registry.NetworkServiceEndpoint{
				Name:                sandbox.UniqueName("forwarder"),
				NetworkServiceNames: []string{"forwarder"},
			}, sandbox.GenerateTestToken, counterFwd[i])
		}).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counterNse := new(count.Server)
	nse := domain.Nodes[nodeNum-1].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counterNse)

	request := defaultRequest(nsReg.Name)

	counterClient := new(count.Client)
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithAdditionalFunctionality(counterClient))

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	for _, fwd := range domain.Nodes[0].Forwarders {
		fwd.Restart()
	}

	nse.Cancel()

	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	nseReg2.Name += "-2"
	domain.Nodes[nodeNum-1].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, counterNse)

	// Wait for heal to finish successfully
	require.Eventually(t, checkSecondRequestsReceived(counterNse.UniqueRequests), timeout, tick)
	// Client should try to close connection before reselect
	require.Equal(t, 1, counterClient.UniqueCloses())
	// local Forwarder has restarted, new forwarder should not get a Close call
	require.Equal(t, 0, counterFwd[0].Closes())
	if nodeNum > 1 {
		// remote forwarder should get Close
		require.Equal(t, 1, counterFwd[1].Closes())
	}
	require.Equal(t, 0, counterNse.Closes())

	// Refresh shouldn't cause any Close calls
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 0, counterNse.Closes())
	require.Equal(t, 0, counterFwd[0].Closes())
	if nodeNum > 1 {
		require.Equal(t, 1, counterFwd[1].Closes())
	}

	clientCloses := counterClient.Closes()
	// Close should still be able to pass though the whole connection path
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, clientCloses+1, counterClient.Closes())
	require.Equal(t, 1, counterNse.Closes())
	require.Equal(t, 1, counterFwd[0].Closes())
	if nodeNum > 1 {
		require.Equal(t, 2, counterFwd[1].Closes())
	}
}

// If registry died, discover and discoverForwarder elements
// will not be able to query it to get URLs to next app
// but we still expect Close call to finish successfully
func TestReselect_Close_RegistryDied(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// in this test we add counters to apps in chain
	// to make sure that in each app Close call goes through the whole chain,
	// without stopping on an error mid-chain
	counterFwd := new(count.Server)

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNSMgrSupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, _ int) {
			node.NewNSMgr(ctx, "nsmgr", nil, sandbox.GenerateTestToken, nsmgr.NewServer)
			node.NewForwarder(ctx, &registry.NetworkServiceEndpoint{
				Name:                sandbox.UniqueName("forwarder"),
				NetworkServiceNames: []string{"forwarder"},
			}, sandbox.GenerateTestToken, counterFwd)
		}).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counterNse := new(count.Server)
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counterNse)

	request := defaultRequest(nsReg.Name)

	counterClient := new(count.Client)
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithAdditionalFunctionality(counterClient))

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	domain.Registry.Cancel()

	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	require.Equal(t, 1, counterClient.Closes())
	require.Equal(t, 1, counterFwd.Closes())
	require.Equal(t, 1, counterNse.Closes())
}
