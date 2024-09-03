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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	nsclient "github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
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

	nseReg := defaultRegistryEndpoint(nsReg.GetName())

	counter := new(count.Server)
	// allow only one successful request
	inject := injecterror.NewServer(injecterror.WithCloseErrorTimes(), injecterror.WithRequestErrorTimes(1, -1))
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter, inject)

	request := defaultRequest(nsReg.GetName())

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.Requests())

	// fail a request
	request.Connection = conn
	refreshCtx, refreshCancel := context.WithTimeout(ctx, time.Second)
	defer refreshCancel()
	_, err = nsc.Request(refreshCtx, request.Clone())
	require.Error(t, err)

	// Close will reach the endpoint from healing
	require.Eventually(t, func() bool {
		return counter.Closes() == 1
	}, timeout, tick)
	_, err = nsc.Close(ctx, conn.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.Closes())
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

	nseReg := defaultRegistryEndpoint(nsReg.GetName())

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

	request := defaultRequest(nsReg.GetName())

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, skipCount+1, counter.UniqueRequests())
	require.Equal(t, skipCount+1, counter.Requests())

	selectedForwarder := conn.GetPath().GetPathSegments()[2].GetName()

	requestsCount := counter.Requests()
	for i := 0; i < reselectCount; i++ {
		_, err = nsc.Close(ctx, conn)
		require.NoError(t, err)

		// check that we select a different forwarder
		selectedForwarder = conn.GetPath().GetPathSegments()[2].GetName()
		request.Connection = conn
		conn, err = nsc.Request(ctx, request.Clone())
		require.NoError(t, err)
		require.Equal(t, skipCount+1, counter.UniqueRequests())
		require.Equal(t, requestsCount+3, counter.Requests())
		requestsCount = counter.Requests()
		if selectedForwarder != conn.GetPath().GetPathSegments()[2].GetName() {
			break
		}
	}
	require.NotEqual(t, selectedForwarder, conn.GetPath().GetPathSegments()[2].GetName())
}

func Test_DiscoverForwarder_ChangeForwarderOnDeath_LostHeal(t *testing.T) {
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

	nseReg := defaultRegistryEndpoint(nsReg.GetName())

	counter := new(count.Server)
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.GetName())

	clientCounter := new(count.Client)
	// make sure that Close from heal doesn't clear the forwarder name
	// we want to clear it automatically in discoverforwarder element on Request
	clientInject := injecterror.NewClient(injecterror.WithRequestErrorTimes(), injecterror.WithCloseErrorTimes(-1))
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken,
		nsclient.WithAdditionalFunctionality(clientCounter, clientInject))

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 1, counter.Requests())

	selectedForwarder := conn.GetPath().GetPathSegments()[2].GetName()

	domain.Nodes[0].Forwarders[selectedForwarder].Cancel()

	require.Eventually(t, checkSecondRequestsReceived(counter.Requests), timeout, tick)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 2, counter.Requests())
	require.Equal(t, 1, counter.Closes())

	// check different forwarder selected
	request.Connection = conn
	conn, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 3, counter.Requests())
	require.Equal(t, 1, counter.Closes())
	require.NotEqual(t, selectedForwarder, conn.GetPath().GetPathSegments()[2].GetName())
}

func Test_DiscoverForwarder_ChangeRemoteForwarderOnDeath(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(2).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, _ int) {
			node.NewNSMgr(ctx, "nsmgr", nil, sandbox.GenerateTestToken, nsmgr.NewServer)
		}).
		Build()

	domain.Nodes[0].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
		Name:                sandbox.UniqueName("forwarder-local"),
		NetworkServiceNames: []string{"forwarder"},
	}, sandbox.GenerateTestToken)

	const fwdCount = 10
	for i := 0; i < fwdCount; i++ {
		domain.Nodes[1].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
			Name:                sandbox.UniqueName("forwarder-" + fmt.Sprint(i)),
			NetworkServiceNames: []string{"forwarder"},
		}, sandbox.GenerateTestToken)
	}

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := defaultRegistryService(t.Name())
	nsReg, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.GetName())

	counter := new(count.Server)
	domain.Nodes[1].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.GetName())

	clientCounter := new(count.Client)
	// make sure that Close from heal doesn't clear the forwarder name
	// we want to clear it automatically in discoverforwarder element on Request
	clientInject := injecterror.NewClient(injecterror.WithRequestErrorTimes(), injecterror.WithCloseErrorTimes(-1))
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken,
		nsclient.WithAdditionalFunctionality(clientCounter, clientInject))

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 1, counter.Requests())

	selectedForwarder := conn.GetPath().GetPathSegments()[4].GetName()

	domain.Registry.Restart()

	domain.Nodes[1].Forwarders[selectedForwarder].Cancel()

	require.Eventually(t, checkSecondRequestsReceived(counter.Requests), timeout, tick)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 2, counter.Requests())
	require.Equal(t, 1, counter.Closes())

	// check different forwarder selected
	request.Connection = conn
	conn, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 3, counter.Requests())
	require.Equal(t, 1, counter.Closes())
	require.NotEqual(t, selectedForwarder, conn.GetPath().GetPathSegments()[4].GetName())
}

func Test_DiscoverForwarder_Should_KeepSelectedForwarderWhileConnectionIsFine(t *testing.T) {
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

	nseReg := defaultRegistryEndpoint(nsReg.GetName())

	counter := new(count.Server)
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.GetName())

	var livenessValue atomic.Value
	livenessValue.Store(true)

	var selectedForwarder string

	livenessChecker := func(deadlineCtx context.Context, conn *networkservice.Connection) bool {
		if v := livenessValue.Load().(bool); !v {
			return conn.GetPath().GetPathSegments()[2].GetName() != selectedForwarder
		}
		return true
	}

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken,
		nsclient.WithHealClient(heal.NewClient(ctx,
			heal.WithLivenessCheck(livenessChecker))))

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 1, counter.Requests())

	selectedForwarder = conn.GetPath().GetPathSegments()[2].GetName()

	domain.Registry.Restart()

	domain.Nodes[0].Forwarders[selectedForwarder].Restart()

	require.Eventually(t, checkSecondRequestsReceived(counter.Requests), timeout, tick)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 2, counter.Requests())
	require.Equal(t, 0, counter.Closes())

	request.Connection = conn
	conn, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 3, counter.Requests())
	require.Equal(t, 0, counter.Closes())
	require.Equal(t, selectedForwarder, conn.GetPath().GetPathSegments()[2].GetName())

	// datapath is down
	livenessValue.Store(false)
	domain.Nodes[0].Forwarders[selectedForwarder].Cancel()
	// waiting for healing
	require.Eventually(t, func() bool {
		return counter.Requests() >= 4
	}, timeout, tick)

	request.Connection = conn
	conn, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Greater(t, counter.Closes(), 0)
	require.NotEqual(t, selectedForwarder, conn.GetPath().GetPathSegments()[2].GetName())
}
