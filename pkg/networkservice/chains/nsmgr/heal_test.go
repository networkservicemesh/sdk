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
	"net"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	tick         = 10 * time.Millisecond
	timeout      = 10 * time.Second
	tokenTimeout = 5 * time.Second
)

func TestNSMGR_HealEndpoint(t *testing.T) {
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
			name:    "Local New",
			nodeNum: 0,
		},
		{
			name:     "Remote Restored",
			nodeNum:  1,
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
			testNSMGRHealEndpoint(t, sample.nodeNum, sample.restored)
		})
	}
}

func testNSMGRHealEndpoint(t *testing.T, nodeNum int, restored bool) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(nodeNum + 1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsReg, err := domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)
	if !restored {
		nseReg.ExpirationTime = timestamppb.New(time.Now().Add(tokenTimeout))
	}

	nseCtx, nseCtxCancel := context.WithCancel(context.Background())

	counter := &counterServer{}

	var nse *sandbox.EndpointEntry
	if restored {
		nse = domain.Nodes[nodeNum].NewEndpoint(nseCtx, nseReg, sandbox.GenerateTestToken, counter)
	} else {
		nse = domain.Nodes[nodeNum].NewEndpoint(nseCtx, nseReg, sandbox.GenerateExpiringToken(tokenTimeout), counter)
	}

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 2+3*(nodeNum+1), len(conn.Path.PathSegments))

	nseCtxCancel()

	nseReg2 := defaultRegistryEndpoint(nsReg.Name)
	if restored {
		// Wait grpc unblock the port
		require.Eventually(t, checkURLFree(nse.URL), timeout, tick)

		nseReg2.Name = nse.Name
		nseReg2.Url = nse.URL.String()
	} else {
		nseReg2.Name += "-2"
	}
	domain.Nodes[nodeNum].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, counter)

	if restored {
		require.Eventually(t, checkSecondRequestsReceived(func() int {
			return int(atomic.LoadInt32(&counter.Requests))
		}), timeout, tick)
		require.Equal(t, int32(2), atomic.LoadInt32(&counter.Requests))
	} else {
		require.Eventually(t, checkSecondRequestsReceived(counter.UniqueRequests), timeout, tick)
		require.Equal(t, 2, counter.UniqueRequests())
	}

	// Check refresh
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// Close.
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	if restored {
		require.Equal(t, int32(3), atomic.LoadInt32(&counter.Requests))
	} else {
		require.Equal(t, 2, counter.UniqueRequests())
	}
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Closes))
}

func TestNSMGR_HealForwarder(t *testing.T) {
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
			name:    "Local New",
			nodeNum: 0,
		},
		{
			name:     "Remote Restored",
			nodeNum:  1,
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
			testNSMGRHealForwarder(t, sample.nodeNum, sample.restored)
		})
	}
}

func testNSMGRHealForwarder(t *testing.T, nodeNum int, restored bool) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var forwarderCtxCancel context.CancelFunc
	var forwarder *sandbox.EndpointEntry
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(2).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, i int) {
			if i != nodeNum {
				sandbox.SetupDefaultNode(ctx, node, nsmgr.NewServer)
			} else {
				forwarderCtxCancel, forwarder = setupCancelableForwarderNode(ctx, node, restored)
			}
		}).
		Build()

	nsReg, err := domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	counter := &counterServer{}
	domain.Nodes[1].NewEndpoint(ctx, defaultRegistryEndpoint(nsReg.Name), sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 8, len(conn.Path.PathSegments))

	forwarderCtxCancel()

	forwarderReg := &registry.NetworkServiceEndpoint{
		Name: sandbox.Name("forwarder-2"),
	}
	if restored {
		// Wait grpc unblock the port
		require.Eventually(t, checkURLFree(forwarder.URL), timeout, tick)

		forwarderReg.Name = forwarder.Name
		forwarderReg.Url = forwarder.URL.String()
	}
	domain.Nodes[nodeNum].NewForwarder(ctx, forwarderReg, sandbox.GenerateTestToken)

	if restored {
		require.Eventually(t, checkSecondRequestsReceived(func() int {
			return int(atomic.LoadInt32(&counter.Requests))
		}), timeout, tick)
		require.Equal(t, int32(2), atomic.LoadInt32(&counter.Requests))
	} else {
		require.Eventually(t, checkSecondRequestsReceived(counter.UniqueRequests), timeout, tick)
		require.Equal(t, 2, counter.UniqueRequests())
	}

	// Check refresh
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// Close.
	closes := atomic.LoadInt32(&counter.Closes)
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	if restored {
		require.Equal(t, int32(3), atomic.LoadInt32(&counter.Requests))
		require.Equal(t, closes+1, atomic.LoadInt32(&counter.Closes))
	} else {
		require.Equal(t, 2, counter.UniqueRequests())
		require.Equal(t, 2, counter.UniqueCloses())
	}
}

func setupCancelableForwarderNode(ctx context.Context, node *sandbox.Node, restored bool) (context.CancelFunc, *sandbox.EndpointEntry) {
	node.NewNSMgr(ctx, sandbox.Name("nsmgr"), nil, sandbox.GenerateTestToken, nsmgr.NewServer)

	forwarderCtx, forwarderCtxCancel := context.WithCancel(ctx)

	forwarderReg := &registry.NetworkServiceEndpoint{
		Name: sandbox.Name("forwarder"),
	}

	var forwarder *sandbox.EndpointEntry
	if restored {
		forwarder = node.NewForwarder(forwarderCtx, forwarderReg, sandbox.GenerateTestToken)
	} else {
		forwarder = node.NewForwarder(forwarderCtx, forwarderReg, sandbox.GenerateExpiringToken(tokenTimeout))
	}

	return forwarderCtxCancel, forwarder
}

func TestNSMGR_HealRemoteNSMgrRestored(t *testing.T) {
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
		{
			name:     "Remote Restored",
			nodeNum:  1,
			restored: true,
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
	if restored {
		// Restore NSMgr test cases cannot work without registry healing.
		t.Skip("https://github.com/networkservicemesh/sdk/issues/713")
	}

	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var nsmgrCtxCancel context.CancelFunc
	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(3).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, i int) {
			if i != nodeNum {
				sandbox.SetupDefaultNode(ctx, node, nsmgr.NewServer)
			} else {
				nsmgrCtxCancel = setupCancellableNSMgrNode(ctx, node, restored)
			}
		}).
		Build()

	nsReg, err := domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	counter := &counterServer{}
	domain.Nodes[1].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	request := defaultRequest(nsReg.Name)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 8, len(conn.Path.PathSegments))

	if !restored {
		nseReg2 := defaultRegistryEndpoint(nsReg.Name)
		nseReg2.Name += "-2"

		domain.Nodes[2].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, counter)
	}

	nsmgrCtxCancel()

	if restored {
		mgr := domain.Nodes[nodeNum].NSMgr
		require.Eventually(t, checkURLFree(mgr.URL), timeout, tick)

		domain.Nodes[nodeNum].NewNSMgr(ctx, mgr.Name, mgr.URL, sandbox.GenerateTestToken, nsmgr.NewServer)
	}

	if restored {
		require.Eventually(t, checkSecondRequestsReceived(func() int {
			return int(atomic.LoadInt32(&counter.Requests))
		}), timeout, tick)
		require.Equal(t, int32(2), atomic.LoadInt32(&counter.Requests))
	} else {
		require.Eventually(t, checkSecondRequestsReceived(counter.UniqueRequests), timeout, tick)
		require.Equal(t, 2, counter.UniqueRequests())
	}

	// Check refresh
	request.Connection = conn
	_, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)

	// Close.
	closes := atomic.LoadInt32(&counter.Closes)
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	if restored {
		require.Equal(t, int32(3), atomic.LoadInt32(&counter.Requests))
		require.Equal(t, closes+1, atomic.LoadInt32(&counter.Closes))
	} else {
		require.Equal(t, 2, counter.UniqueRequests())
		require.Equal(t, 2, counter.UniqueCloses())
	}
}

func setupCancellableNSMgrNode(ctx context.Context, node *sandbox.Node, restored bool) context.CancelFunc {
	nsmgrCtx, nsmgrCtxCancel := context.WithCancel(ctx)

	nsmgrName := sandbox.Name("nsmgr")
	if restored {
		node.NewNSMgr(nsmgrCtx, nsmgrName, nil, sandbox.GenerateTestToken, nsmgr.NewServer)
	} else {
		node.NewNSMgr(nsmgrCtx, nsmgrName, nil, sandbox.GenerateExpiringToken(tokenTimeout), nsmgr.NewServer)
	}

	node.NewForwarder(ctx, &registry.NetworkServiceEndpoint{
		Name: sandbox.Name("forwarder"),
	}, sandbox.GenerateTestToken)

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

	nsReg, err := domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseCtx, nseCtxCancel := context.WithCancel(ctx)

	domain.Nodes[0].NewEndpoint(nseCtx, defaultRegistryEndpoint(nsReg.Name), sandbox.GenerateTestToken)

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

func checkURLFree(u *url.URL) func() bool {
	return func() bool {
		ln, err := net.Listen("tcp", grpcutils.URLToTarget(u))
		if err != nil {
			return false
		}
		err = ln.Close()
		return err == nil
	}
}
