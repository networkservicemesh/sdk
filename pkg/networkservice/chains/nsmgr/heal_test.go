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

// Package nsmgr_test define a tests for NSMGR chain element.
package nsmgr_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	tick    = 10 * time.Millisecond
	timeout = 10 * time.Second
)

func TestNSMGRHealEndpoint(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()

	nsReg, err := domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	nseCtx, nseCtxCancel := context.WithCancel(context.Background())

	counter := &counterServer{}
	_, err = domain.Nodes[0].NewEndpoint(nseCtx, nseReg, sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[1].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())

	nseCtxCancel()

	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	nseReg2.Name += "-2"
	_, err = domain.Nodes[0].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

	// Wait reconnecting to the new NSE
	require.Eventually(t, checkSecondRequestsReceived(counter.UniqueRequests), timeout, tick)
	require.Equal(t, 2, counter.UniqueRequests())

	// Check refresh
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// Close with old connection
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	require.Equal(t, 2, counter.UniqueRequests())
	require.Equal(t, 1, counter.UniqueCloses())
}

func TestNSMGR_HealLocalForwarder(t *testing.T) {
	forwarderCtx, forwarderCtxCancel := context.WithCancel(context.Background())
	defer forwarderCtxCancel()

	customConfig := []*sandbox.NodeConfig{
		nil,
		{
			ForwarderCtx:               forwarderCtx,
			ForwarderGenerateTokenFunc: sandbox.GenerateTestToken,
		},
	}

	testNSMGRHealForwarder(t, 1, customConfig, forwarderCtxCancel)
}

func TestNSMGR_HealRemoteForwarder(t *testing.T) {
	forwarderCtx, forwarderCtxCancel := context.WithCancel(context.Background())
	defer forwarderCtxCancel()

	customConfig := []*sandbox.NodeConfig{
		{
			ForwarderCtx:               forwarderCtx,
			ForwarderGenerateTokenFunc: sandbox.GenerateTestToken,
		},
	}

	testNSMGRHealForwarder(t, 0, customConfig, forwarderCtxCancel)
}

func testNSMGRHealForwarder(t *testing.T, nodeNum int, customConfig []*sandbox.NodeConfig, forwarderCtxCancel context.CancelFunc) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	builder := sandbox.NewBuilder(t)
	domain := builder.
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		SetCustomConfig(customConfig).
		Build()

	nsReg, err := domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	counter := &counterServer{}
	_, err = domain.Nodes[0].NewEndpoint(ctx, defaultRegistryEndpoint(nsReg.Name), sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[1].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())

	forwarderCtxCancel()

	_, err = domain.Nodes[nodeNum].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
		Name: "forwarder-restored",
	}, sandbox.GenerateTestToken)
	require.NoError(t, err)

	// Wait reconnecting through the new Forwarder
	require.Eventually(t, checkSecondRequestsReceived(counter.UniqueRequests), timeout, tick)
	require.Equal(t, 2, counter.UniqueRequests())

	// Check refresh
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// Close with old connection
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	require.Equal(t, 2, counter.UniqueRequests())
	require.Equal(t, 1, counter.UniqueCloses())
}

func TestNSMGR_HealLocalNSMgrRestored(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	nsmgrCtx, nsmgrCtxCancel := context.WithCancel(ctx)

	customConfig := []*sandbox.NodeConfig{
		nil,
		{
			NsmgrCtx:               nsmgrCtx,
			NsmgrGenerateTokenFunc: sandbox.GenerateTestToken,
		},
	}

	builder := sandbox.NewBuilder(t)
	domain := builder.
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		SetCustomConfig(customConfig).
		Build()

	nsReg, err := domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	counter := &counterServer{}
	_, err = domain.Nodes[0].NewEndpoint(ctx, defaultRegistryEndpoint(nsReg.Name), sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[1].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Requests))

	nsmgrCtxCancel()

	// Wait grpc unblock the port
	require.Eventually(t, func() bool {
		return grpcutils.CheckURLFree(domain.Nodes[1].NSMgr.URL)
	}, timeout, tick)

	restoredNSMgrEntry, restoredNSMgrResources := builder.NewNSMgr(ctx, domain.Nodes[1], domain.Nodes[1].NSMgr.URL.Host, domain.Registry.URL, sandbox.GenerateTestToken)
	domain.Nodes[1].NSMgr = restoredNSMgrEntry
	domain.AddResources(restoredNSMgrResources)

	// Wait NSC restore the connection
	require.Eventually(t, checkSecondRequestsReceived(func() int {
		return int(atomic.LoadInt32(&counter.Requests))
	}), timeout, tick)
	require.Equal(t, int32(2), counter.Requests)
	require.Equal(t, 1, counter.UniqueRequests())

	// Check refresh
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// Close with old connection
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	require.Equal(t, int32(3), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Closes))
}

func TestNSMGR_HealRemoteNSMgr(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	nsmgrCtx, nsmgrCtxCancel := context.WithCancel(ctx)

	customConfig := []*sandbox.NodeConfig{
		{
			NsmgrCtx:               nsmgrCtx,
			NsmgrGenerateTokenFunc: sandbox.GenerateTestToken,
		},
	}

	builder := sandbox.NewBuilder(t)
	domain := builder.
		SetNodesCount(3).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		SetCustomConfig(customConfig).
		Build()

	nsReg, err := domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	counter := &counterServer{}
	_, err = domain.Nodes[0].NewEndpoint(ctx, defaultRegistryEndpoint(nsReg.Name), sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[1].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 8, len(conn.Path.PathSegments))

	nsmgrCtxCancel()

	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	nseReg2.Name += "-2"
	_, err = domain.Nodes[2].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

	// Wait through the new NSMgr
	require.Eventually(t, checkSecondRequestsReceived(counter.UniqueRequests), timeout, tick)
	require.Equal(t, 2, counter.UniqueRequests())

	// Check refresh
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	require.Equal(t, 2, counter.UniqueRequests())
	require.Equal(t, 1, counter.UniqueCloses())
}

func TestNSMGR_CloseHeal(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()

	nsReg, err := domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseCtx, nseCtxCancel := context.WithCancel(ctx)

	_, err = domain.Nodes[0].NewEndpoint(nseCtx, defaultRegistryEndpoint(nsReg.Name), sandbox.GenerateTestToken)
	require.NoError(t, err)

	request := defaultRequest(nsReg.Name)

	nscCtx, nscCtxCancel := context.WithCancel(ctx)

	nsc := domain.Nodes[0].NewClient(nscCtx, sandbox.GenerateTestToken)

	// 1. Request
	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	ignoreCurrent := goleak.IgnoreCurrent()

	// 2. Refresh
	request.Connection = conn

	conn, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// 3. Stop endpoint
	nseCtxCancel()

	// 4. Close connection
	_, _ = nsc.Close(nscCtx, conn.Clone())

	nscCtxCancel()

	require.Eventually(t, func() bool {
		return goleak.Find(ignoreCurrent) == nil
	}, timeout, tick)

	require.NoError(t, ctx.Err())
}

func checkSecondRequestsReceived(requestsDone func() int) func() bool {
	return func() bool {
		return requestsDone() >= 2
	}
}
