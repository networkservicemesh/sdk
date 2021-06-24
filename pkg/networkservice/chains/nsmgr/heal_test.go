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

package nsmgr_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
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

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.DefaultTokenTimeout)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	nseCtx, nseCtxCancel := context.WithCancel(context.Background())

	counter := &counterServer{}
	domain.Nodes[nodeNum].NewEndpoint(nseCtx, nseReg, sandbox.DefaultTokenTimeout, counter)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.DefaultTokenTimeout)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())

	nseCtxCancel()

	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	nseReg2.Name += "-2"
	domain.Nodes[nodeNum].NewEndpoint(ctx, nseReg2, sandbox.DefaultTokenTimeout, counter)

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

	var forwarderCtxCancel context.CancelFunc
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(2).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, nodeN int) {
			if nodeN != nodeNum {
				sandbox.SetupDefaultNode(ctx, node, sandbox.DefaultTokenTimeout, nsmgr.NewServer)
			} else {
				forwarderCtxCancel = setupCancelableForwarderNode(ctx, node)
			}
		}).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.DefaultTokenTimeout)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	counter := &counterServer{}
	domain.Nodes[1].NewEndpoint(ctx, defaultRegistryEndpoint(nsReg.Name), sandbox.DefaultTokenTimeout, counter)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.DefaultTokenTimeout)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())

	forwarderCtxCancel()

	forwarderReg := &registry.NetworkServiceEndpoint{
		Name: sandbox.UniqueName("forwarder-2"),
	}
	domain.Nodes[nodeNum].NewForwarder(ctx, forwarderReg, sandbox.DefaultTokenTimeout)

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

func setupCancelableForwarderNode(ctx context.Context, node *sandbox.Node) context.CancelFunc {
	node.NewNSMgr(ctx, sandbox.UniqueName("nsmgr"), nil, sandbox.DefaultTokenTimeout, nsmgr.NewServer)

	forwarderCtx, forwarderCtxCancel := context.WithCancel(ctx)
	node.NewForwarder(forwarderCtx, &registry.NetworkServiceEndpoint{
		Name: sandbox.UniqueName("forwarder"),
	}, sandbox.DefaultTokenTimeout)

	return forwarderCtxCancel
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

	var nsmgrCtxCancel context.CancelFunc
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(3).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, nodeN int) {
			if nodeN != nodeNum {
				sandbox.SetupDefaultNode(ctx, node, sandbox.DefaultTokenTimeout, nsmgr.NewServer)
			} else {
				nsmgrCtxCancel = setupCancellableNSMgrNode(ctx, node)
			}
		}).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.DefaultTokenTimeout)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counter := &counterServer{}
	domain.Nodes[1].NewEndpoint(ctx, nseReg, sandbox.DefaultTokenTimeout, counter)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.DefaultTokenTimeout)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	if !restored {
		nseReg2 := defaultRegistryEndpoint(nsReg.Name)
		nseReg2.Name += "-2"

		domain.Nodes[2].NewEndpoint(ctx, nseReg2, sandbox.DefaultTokenTimeout, counter)
	}

	nsmgrCtxCancel()

	if restored {
		mgr := domain.Nodes[nodeNum].NSMgr

		// Wait grpc unblock the port
		require.Eventually(t, func() bool {
			return grpcutils.CheckURLFree(domain.Nodes[0].NSMgr.URL)
		}, timeout, tick)

		domain.Nodes[nodeNum].NewNSMgr(ctx, mgr.Name, mgr.URL, sandbox.DefaultTokenTimeout, nsmgr.NewServer)
	}

	if restored {
		// Wait reconnecting through the restored NSMgr
		require.Eventually(t, checkSecondRequestsReceived(func() int {
			return int(atomic.LoadInt32(&counter.Requests))
		}), timeout, tick)
		require.Equal(t, int32(2), atomic.LoadInt32(&counter.Requests))
	} else {
		// Wait reconnecting through the new NSMgr
		require.Eventually(t, checkSecondRequestsReceived(counter.UniqueRequests), timeout, tick)
		require.Equal(t, 2, counter.UniqueRequests())
	}

	// Check refresh
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// Close with old connection
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	if restored {
		require.Equal(t, int32(3), atomic.LoadInt32(&counter.Requests))
		require.Equal(t, int32(1), atomic.LoadInt32(&counter.Closes))
	} else {
		require.Equal(t, 2, counter.UniqueRequests())
		require.Equal(t, 1, counter.UniqueCloses())
	}
}

func setupCancellableNSMgrNode(ctx context.Context, node *sandbox.Node) context.CancelFunc {
	nsmgrCtx, nsmgrCtxCancel := context.WithCancel(ctx)

	node.NewNSMgr(nsmgrCtx, sandbox.UniqueName("nsmgr"), nil, sandbox.DefaultTokenTimeout, nsmgr.NewServer)

	node.NewForwarder(ctx, &registry.NetworkServiceEndpoint{
		Name: sandbox.UniqueName("forwarder"),
	}, sandbox.DefaultTokenTimeout)

	return nsmgrCtxCancel
}

func TestNSMGR_CloseHeal(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.DefaultTokenTimeout)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseCtx, nseCtxCancel := context.WithCancel(ctx)

	domain.Nodes[0].NewEndpoint(nseCtx, defaultRegistryEndpoint(nsReg.Name), sandbox.DefaultTokenTimeout)

	request := defaultRequest(nsReg.Name)

	nscCtx, nscCtxCancel := context.WithCancel(ctx)

	nsc := domain.Nodes[0].NewClient(nscCtx, sandbox.DefaultTokenTimeout)

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
