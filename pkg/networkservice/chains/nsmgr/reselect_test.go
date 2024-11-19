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
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

// Even if NSMgr has restarted,
// we expect that all other apps should get a Close call
func TestReselect_NsmgrRestart(t *testing.T) {
	var samples = []struct {
		name          string
		nodeNum       int
		restartLocal  bool
		restartRemote bool
	}{
		{
			name:    "Local",
			nodeNum: 1,
		},
		{
			name:         "Remote_RestartLocal",
			nodeNum:      2,
			restartLocal: true,
		},
		{
			name:          "Remote_RestartRemote",
			nodeNum:       2,
			restartRemote: true,
		},
		{
			name:          "Remote_RestartBoth",
			nodeNum:       2,
			restartLocal:  true,
			restartRemote: true,
		},
	}

	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			testReselectWithNsmgrRestart(t, sample.nodeNum, sample.restartLocal, sample.restartRemote)
		})
	}
}

func testReselectWithNsmgrRestart(t *testing.T, nodeNum int, restartLocal, restartRemote bool) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// in this test we add counters to apps in chain
	// to make sure that in each app Close call goes through the whole chain,
	// without stopping on an error mid-chain
	var counterFwd []*count.Client
	for i := 0; i < nodeNum; i++ {
		counterFwd = append(counterFwd, new(count.Client))
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
			}, sandbox.GenerateTestToken, sandbox.WithForwarderAdditionalFunctionalityClient(counterFwd[i]))
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

	if restartLocal {
		domain.Nodes[0].NSMgr.Restart()
	}
	if restartRemote {
		domain.Nodes[1].NSMgr.Restart()
	}

	nse.Cancel()

	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	nseReg2.Name += "-2"
	domain.Nodes[nodeNum-1].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, counterNse)

	// Wait for heal to finish successfully
	require.Eventually(t, checkSecondRequestsReceived(counterNse.UniqueRequests), timeout, tick)
	// Client should try to close connection before reselect
	require.Equal(t, 1, counterClient.UniqueCloses())
	// Forwarder(s) should get a Close, even though NSMgr(s) restarted and didn't pass the Close
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

// Even if Local forwarder has restarted,
// we expect that all other apps should get a Close call.
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
	var counterFwd []*count.Client
	for i := 0; i < nodeNum; i++ {
		counterFwd = append(counterFwd, new(count.Client))
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
			}, sandbox.GenerateTestToken, sandbox.WithForwarderAdditionalFunctionalityClient(counterFwd[i]))
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

	// Kill a local forwarder and NSE
	for _, fwd := range domain.Nodes[0].Forwarders {
		fwd.Cancel()
	}
	nse.Cancel()

	// Restart NSE and local forwarder
	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	nseReg2.Name += "-2"
	domain.Nodes[nodeNum-1].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, counterNse)

	for _, fwd := range domain.Nodes[0].Forwarders {
		fwd.Restart()
	}

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

// If registry died, NSMgr and Forwarder
// will not be able to query it to get URLs to next app
// but we still expect Close call to finish successfully
func TestReselect_Close_RegistryDied(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// in this test we add counters to apps in chain
	// to make sure that in each app Close call goes through the whole chain,
	// without stopping on an error mid-chain
	counterFwd := new(count.Client)

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
			}, sandbox.GenerateTestToken, sandbox.WithForwarderAdditionalFunctionalityClient(counterFwd))
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
