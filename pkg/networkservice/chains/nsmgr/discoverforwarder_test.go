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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"

	nsclient "github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func Test_DiscoverForwarder_CloseAfterError(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := defaultRegistryService(t.Name())
	nsReg, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counter := new(count.Server)
	// allow only one successful request
	inject := injecterror.NewServer(injecterror.WithCloseErrorTimes(), injecterror.WithRequestErrorTimes(1, -1))
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter, inject)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.Requests())

	fmt.Println("nacskq: root request with fail")
	// fail a request
	request.Connection = conn
	refreshCtx, refreshCancel := context.WithTimeout(ctx, time.Second)
	defer refreshCancel()
	_, err = nsc.Request(refreshCtx, request.Clone())
	require.Error(t, err)

	fmt.Println("nacskq: root close")
	// check that closes still reach the NSE
	require.Equal(t, 0, counter.Closes())
	_, err = nsc.Close(ctx, conn.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.Closes())
}

func Test_DiscoverForwarder_KeepForwarderOnErrors(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, _ int) {
			node.NewNSMgr(ctx, "nsmgr", nil, sandbox.GenerateTestToken, nsmgr.NewServer)
		}).
		Build()

	const fwdCount = 10
	for i := 0; i < fwdCount; i++ {
		domain.Nodes[0].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
			Name:                sandbox.UniqueName("forwarder-" + fmt.Sprint(i)),
			NetworkServiceNames: []string{"forwarder"},
		}, sandbox.GenerateTestToken)
	}

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := defaultRegistryService(t.Name())
	nsReg, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	errorIndices := []int{}
	// skip half of the available forwarders
	skipCount := fwdCount / 2
	for i := 0; i < skipCount; i++ {
		errorIndices = append(errorIndices, i)
	}
	// then allow 1 successful request, then 1 error
	errorIndices = append(errorIndices, errorIndices[len(errorIndices)-1]+2) //nolint:gocritic
	// then allow 1 successful request, then 3 errors
	errorIndices = append(errorIndices,
		errorIndices[len(errorIndices)-1]+2,
		errorIndices[len(errorIndices)-1]+3,
		errorIndices[len(errorIndices)-1]+4,
	)
	inject := injecterror.NewServer(injecterror.WithCloseErrorTimes(), injecterror.WithRequestErrorTimes(errorIndices...))
	counter := new(count.Server)
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter, inject)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, skipCount+1, counter.UniqueRequests())
	require.Equal(t, skipCount+1, counter.Requests())

	selectedFwd := conn.GetPath().GetPathSegments()[2].Name

	// check forwarder doesn't change after 1 error
	request.Connection = conn
	conn, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, skipCount+1, counter.UniqueRequests())
	require.Equal(t, skipCount+3, counter.Requests())
	require.Equal(t, selectedFwd, conn.GetPath().GetPathSegments()[2].Name)

	// check forwarder doesn't change after 3 consecutive errors
	request.Connection = conn
	conn, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, skipCount+1, counter.UniqueRequests())
	require.Equal(t, skipCount+7, counter.Requests())
	require.Equal(t, selectedFwd, conn.GetPath().GetPathSegments()[2].Name)
}

func Test_DiscoverForwarder_KeepForwarderOnNSEDeath_NoHeal(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout*1000)

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, _ int) {
			node.NewNSMgr(ctx, "nsmgr", nil, sandbox.GenerateTestToken, nsmgr.NewServer)
		}).
		Build()

	const fwdCount = 10
	for i := 0; i < fwdCount; i++ {
		domain.Nodes[0].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
			Name:                sandbox.UniqueName("forwarder-" + fmt.Sprint(i)),
			NetworkServiceNames: []string{"forwarder"},
		}, sandbox.GenerateTestToken)
	}

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := defaultRegistryService(t.Name())
	nsReg, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counter := new(count.Server)
	nse := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, nsclient.WithHealClient(null.NewClient()))

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 1, counter.Requests())

	selectedFwd := conn.GetPath().GetPathSegments()[2].Name

	fmt.Println("nacskq: root cancel nse")
	nse.Cancel()

	// fail a refresh on connection timeout
	refreshCtx, refreshCancel := context.WithTimeout(ctx, time.Second)
	defer refreshCancel()
	request.Connection = conn
	_, err = nsc.Request(refreshCtx, request.Clone())
	require.Error(t, err)

	fmt.Println("nacskq: root after failed request")

	// create a new NSE
	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	nseReg2.Name += "-2"
	counter2 := new(count.Server)
	// inject 1 error to make sure that don't go the "first try forwarder in path" route
	inject2 := injecterror.NewServer(injecterror.WithCloseErrorTimes(), injecterror.WithRequestErrorTimes(0, 1))
	regEntry2 := domain.Nodes[0].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, counter2, inject2)

	fmt.Println("nacskq: root new request")

	// check that forwarder doesn't change after NSE re-selction
	request.Connection = conn
	request.Connection.NetworkServiceEndpointName = ""
	conn, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 3, counter2.UniqueRequests())
	require.Equal(t, 3, counter2.Requests())
	require.Equal(t, regEntry2.Name, conn.GetPath().GetPathSegments()[3].Name)
	require.Equal(t, selectedFwd, conn.GetPath().GetPathSegments()[2].Name)
}

func Test_DiscoverForwarder_ChangeForwarderOnDeath_NoHeal(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, _ int) {
			node.NewNSMgr(ctx, "nsmgr", nil, sandbox.GenerateTestToken, nsmgr.NewServer)
		}).
		Build()

	const fwdCount = 10
	for i := 0; i < fwdCount; i++ {
		domain.Nodes[0].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
			Name:                sandbox.UniqueName("forwarder-" + fmt.Sprint(i)),
			NetworkServiceNames: []string{"forwarder"},
		}, sandbox.GenerateTestToken)
	}

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := defaultRegistryService(t.Name())
	nsReg, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counter := new(count.Server)
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, nsclient.WithHealClient(null.NewClient()))

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 1, counter.Requests())

	selectedFwd := conn.GetPath().GetPathSegments()[2].Name

	fmt.Println("nacskq: root cancel fwd")
	domain.Nodes[0].Forwarders[selectedFwd].Cancel()

	fmt.Println("nacskq: root request different fwd")
	// check different forwarder selected
	request.Connection = conn
	conn, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 2, counter.Requests())
	require.NotEqual(t, selectedFwd, conn.GetPath().GetPathSegments()[2].Name)
}

func Test_DiscoverForwarder_ChangeForwarderOnClose(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, _ int) {
			node.NewNSMgr(ctx, "nsmgr", nil, sandbox.GenerateTestToken, nsmgr.NewServer)
		}).
		Build()

	const fwdCount = 10
	for i := 0; i < fwdCount; i++ {
		domain.Nodes[0].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
			Name:                sandbox.UniqueName("forwarder-" + fmt.Sprint(i)),
			NetworkServiceNames: []string{"forwarder"},
		}, sandbox.GenerateTestToken)
	}

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := defaultRegistryService(t.Name())
	nsReg, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	// forwarder selection is stochastic
	// it's possible to get the same forwarder after close by pure luck
	// so we try re-selecting it several times
	const reselectCount = 10

	errorIndices := []int{}
	// skip half of the available forwarders
	skipCount := fwdCount / 2
	for i := 0; i < skipCount; i++ {
		errorIndices = append(errorIndices, i)
	}
	// same pattern for each re-selection attempt
	for i := 0; i < reselectCount; i++ {
		// allow one successful request, then two errors
		errorIndices = append(errorIndices, errorIndices[len(errorIndices)-1]+2, errorIndices[len(errorIndices)-1]+3)
	}
	// then allow one successful request, then two errors
	// errorIndices = append(errorIndices, errorIndices[len(errorIndices)-1]+2, errorIndices[len(errorIndices)-1]+3)
	inject := injecterror.NewServer(injecterror.WithCloseErrorTimes(), injecterror.WithRequestErrorTimes(errorIndices...))
	counter := new(count.Server)
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter, inject)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, skipCount+1, counter.UniqueRequests())
	require.Equal(t, skipCount+1, counter.Requests())

	selectedFwd := conn.GetPath().GetPathSegments()[2].Name

	requestsCount := counter.Requests()
	for i := 0; i < reselectCount; i++ {
		_, err = nsc.Close(ctx, conn)
		require.NoError(t, err)

		// check that we select a different forwarder
		selectedFwd = conn.GetPath().GetPathSegments()[2].Name
		request.Connection = conn
		conn, err = nsc.Request(ctx, request.Clone())
		require.NoError(t, err)
		require.Equal(t, skipCount+1, counter.UniqueRequests())
		require.Equal(t, requestsCount+3, counter.Requests())
		requestsCount = counter.Requests()
		if selectedFwd != conn.GetPath().GetPathSegments()[2].Name {
			break
		}
	}
	require.NotEqual(t, selectedFwd, conn.GetPath().GetPathSegments()[2].Name)
}
