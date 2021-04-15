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
	"fmt"
	"net"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func TestNSMGR_RemoteUsecase_Parallel(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()

	counter := &counterServer{}

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service-remote",
			Context:        &networkservice.ConnectionContext{},
		},
	}
	go func() {
		time.Sleep(time.Millisecond * 100)
		nseReg := &registry.NetworkServiceEndpoint{
			Name:                "final-endpoint",
			NetworkServiceNames: []string{"my-service-remote"},
		}
		_, err := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)
		require.NoError(t, err)
	}()
	nsc := domain.Nodes[1].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))
	require.Equal(t, int32(2), atomic.LoadInt32(&counter.Requests))

	// Close.

	e, err := nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.NotNil(t, e)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Closes))
}

func TestNSMGR_SelectsRestartingEndpoint(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "ns-1",
			Context:        &networkservice.ConnectionContext{},
		},
	}

	// 1. Start listen address and register endpoint
	netListener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer func() { _ = netListener.Close() }()

	_, err = domain.Nodes[0].NSRegistryClient.Register(ctx, &registry.NetworkService{
		Name:    "ns-1",
		Payload: payload.IP,
	})
	require.NoError(t, err)

	_, err = domain.Nodes[0].EndpointRegistryClient.Register(ctx, &registry.NetworkServiceEndpoint{
		Name:                "nse-1",
		NetworkServiceNames: []string{"ns-1"},
		Url:                 "tcp://" + netListener.Addr().String(),
	})
	require.NoError(t, err)

	// 2. Postpone endpoint start
	time.AfterFunc(time.Second, func() {
		s := grpc.NewServer()
		endpoint.NewServer(ctx, sandbox.GenerateTestToken).Register(s)
		_ = s.Serve(netListener)
	})

	// 3. Create client and request endpoint
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 5, len(conn.Path.PathSegments))

	require.NoError(t, ctx.Err())
}

func TestNSMGR_SettingPayload(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "ns-1",
			Context:        &networkservice.ConnectionContext{},
		},
	}

	// 1. Start listen address and register endpoint
	netListener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer func() { _ = netListener.Close() }()

	_, err = domain.Nodes[0].NSRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "ns-1",
	})
	require.NoError(t, err)

	_, err = domain.Nodes[0].EndpointRegistryClient.Register(ctx, &registry.NetworkServiceEndpoint{
		Name:                "nse-1",
		NetworkServiceNames: []string{"ns-1"},
		Url:                 "tcp://" + netListener.Addr().String(),
	})
	require.NoError(t, err)

	// 2. Postpone endpoint start
	time.AfterFunc(time.Second, func() {
		s := grpc.NewServer()
		endpoint.NewServer(ctx, sandbox.GenerateTestToken).Register(s)
		_ = s.Serve(netListener)
	})

	// 3. Create client and request endpoint
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, payload.IP, conn.Payload)

	require.NoError(t, ctx.Err())
}

func TestNSMGR_RemoteUsecase_BusyEndpoints(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()

	counter := new(counterServer)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service-remote",
			Context:        &networkservice.ConnectionContext{},
		},
	}
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			nseReg := &registry.NetworkServiceEndpoint{
				Name:                "final-endpoint-" + strconv.Itoa(id),
				NetworkServiceNames: []string{"my-service-remote"},
			}
			_, err := domain.Nodes[1].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, newBusyEndpoint())
			require.NoError(t, err)
			wg.Done()
		}(i)
	}
	go func() {
		wg.Wait()
		time.Sleep(time.Second / 2)
		nseReg := &registry.NetworkServiceEndpoint{
			Name:                "final-endpoint-3",
			NetworkServiceNames: []string{"my-service-remote"},
		}
		_, err := domain.Nodes[1].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)
		require.NoError(t, err)
	}()
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(2), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 8, len(conn.Path.PathSegments))
}

func TestNSMGR_RemoteUsecase(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{"my-service-remote"},
	}

	counter := &counterServer{}
	_, err := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service-remote",
			Context:        &networkservice.ConnectionContext{},
		},
	}

	nsc := domain.Nodes[1].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(2), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Close.

	e, err := nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.NotNil(t, e)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Closes))
}

func TestNSMGR_ConnectToDeadNSE(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetContext(ctx).
		SetRegistryProxySupplier(nil).
		Build()

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{"my-service-remote"},
	}

	counter := &counterServer{}

	nseCtx, killNse := context.WithCancel(ctx)
	_, err := domain.Nodes[0].NewEndpoint(nseCtx, nseReg, sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service-remote",
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 5, len(conn.Path.PathSegments))

	killNse()
	// Simulate refresh from client.
	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	_, err = nsc.Request(ctx, refreshRequest)
	require.Error(t, err)
	require.NoError(t, ctx.Err())
}

func TestNSMGR_LocalUsecase(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetContext(ctx).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{"my-service-remote"},
	}

	counter := &counterServer{}
	_, err := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service-remote",
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 5, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn2, err := nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn2)
	require.Equal(t, 5, len(conn2.Path.PathSegments))
	require.Equal(t, int32(2), atomic.LoadInt32(&counter.Requests))
	// Close.

	e, err := nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.NotNil(t, e)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Closes))
}

func TestNSMGR_PassThroughRemote(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	const nodesCount = 7
	domain := sandbox.NewBuilder(t).
		SetNodesCount(nodesCount).
		SetContext(ctx).
		SetRegistryProxySupplier(nil).
		Build()

	for i := 0; i < nodesCount; i++ {
		var additionalFunctionality []networkservice.NetworkServiceServer
		if i != 0 {
			// Passtrough to the node i-1
			additionalFunctionality = []networkservice.NetworkServiceServer{
				chain.NewNetworkServiceServer(
					clienturl.NewServer(domain.Nodes[i].NSMgr.URL),
					connect.NewServer(ctx,
						sandbox.NewCrossConnectClientFactory(sandbox.GenerateTestToken,
							newPassTroughClient(fmt.Sprintf("my-service-remote-%v", i-1)),
							kernel.NewClient()),
						connect.WithDialOptions(sandbox.DefaultDialOptions(sandbox.GenerateTestToken)...),
					),
				),
			}
		}
		nseReg := &registry.NetworkServiceEndpoint{
			Name:                fmt.Sprintf("endpoint-%v", i),
			NetworkServiceNames: []string{fmt.Sprintf("my-service-remote-%v", i)},
		}
		_, err := domain.Nodes[i].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, additionalFunctionality...)
		require.NoError(t, err)
	}

	nsc := domain.Nodes[nodesCount-1].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: fmt.Sprintf("my-service-remote-%v", nodesCount-1),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length to first endpoint is 5
	// Path length from NSE client to other remote endpoint is 8
	require.Equal(t, 8*(nodesCount-1)+5, len(conn.Path.PathSegments))
}

func TestNSMGR_PassThroughLocal(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	const nsesCount = 7
	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetContext(ctx).
		SetRegistryProxySupplier(nil).
		Build()

	for i := 0; i < nsesCount; i++ {
		var additionalFunctionality []networkservice.NetworkServiceServer
		if i != 0 {
			additionalFunctionality = []networkservice.NetworkServiceServer{
				chain.NewNetworkServiceServer(
					clienturl.NewServer(domain.Nodes[0].NSMgr.URL),
					connect.NewServer(ctx,
						sandbox.NewCrossConnectClientFactory(sandbox.GenerateTestToken,
							newPassTroughClient(fmt.Sprintf("my-service-remote-%v", i-1)),
							kernel.NewClient()),
						connect.WithDialOptions(sandbox.DefaultDialOptions(sandbox.GenerateTestToken)...),
					),
				),
			}
		}
		nseReg := &registry.NetworkServiceEndpoint{
			Name:                fmt.Sprintf("endpoint-%v", i),
			NetworkServiceNames: []string{fmt.Sprintf("my-service-remote-%v", i)},
		}
		_, err := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, additionalFunctionality...)
		require.NoError(t, err)
	}

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: fmt.Sprintf("my-service-remote-%v", nsesCount-1),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length to first endpoint is 5
	// Path length from NSE client to other local endpoint is 5
	require.Equal(t, 5*(nsesCount-1)+5, len(conn.Path.PathSegments))
}

func TestNSMGR_ShouldCorrectlyAddForwardersWithSameNames(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(nil).
		SetRegistryExpiryDuration(sandbox.RegistryExpiryDuration).
		SetContext(ctx).
		Build()

	forwarderReg := &registry.NetworkServiceEndpoint{
		Name: "forwarder",
	}

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "endpoint",
		NetworkServiceNames: []string{"service"},
	}

	// 1. Add forwarders
	forwarder1Reg := forwarderReg.Clone()
	_, err := domain.Nodes[0].NewForwarder(ctx, forwarder1Reg, sandbox.GenerateTestToken)
	require.NoError(t, err)

	forwarder2Reg := forwarderReg.Clone()
	_, err = domain.Nodes[0].NewForwarder(ctx, forwarder2Reg, sandbox.GenerateTestToken)
	require.NoError(t, err)

	forwarder3Reg := forwarderReg.Clone()
	_, err = domain.Nodes[0].NewForwarder(ctx, forwarder3Reg, sandbox.GenerateTestToken)
	require.NoError(t, err)

	// 2. Wait for refresh
	<-time.After(sandbox.RegistryExpiryDuration)

	testNSEAndClient(ctx, t, domain, nseReg.Clone())

	// 3. Delete first forwarder
	_, err = domain.Nodes[0].ForwarderRegistryClient.Unregister(ctx, forwarder1Reg)
	require.NoError(t, err)

	testNSEAndClient(ctx, t, domain, nseReg.Clone())

	// 4. Delete last forwarder
	_, err = domain.Nodes[0].ForwarderRegistryClient.Unregister(ctx, forwarder3Reg)
	require.NoError(t, err)

	testNSEAndClient(ctx, t, domain, nseReg.Clone())

	_, err = domain.Nodes[0].ForwarderRegistryClient.Unregister(ctx, forwarder2Reg)
	require.NoError(t, err)
}

func TestNSMGR_ShouldCorrectlyAddEndpointsWithSameNames(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetRegistryExpiryDuration(sandbox.RegistryExpiryDuration).
		SetContext(ctx).
		Build()

	nseReg := &registry.NetworkServiceEndpoint{
		Name: "endpoint",
	}

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	// 1. Add endpoints
	nse1Reg := nseReg.Clone()
	nse1Reg.NetworkServiceNames = []string{"service-1"}
	_, err := domain.Nodes[0].NewEndpoint(ctx, nse1Reg, sandbox.GenerateTestToken)
	require.NoError(t, err)

	nse2Reg := nseReg.Clone()
	nse2Reg.NetworkServiceNames = []string{"service-2"}
	_, err = domain.Nodes[0].NewEndpoint(ctx, nse2Reg, sandbox.GenerateTestToken)
	require.NoError(t, err)

	// 2. Wait for refresh
	<-time.After(sandbox.RegistryExpiryDuration)

	// 3. Request
	_, err = nsc.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "service-1",
		},
	})
	require.NoError(t, err)

	_, err = nsc.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "service-2",
		},
	})
	require.NoError(t, err)

	// 3. Delete endpoints
	_, err = domain.Nodes[0].ForwarderRegistryClient.Unregister(ctx, nse1Reg)
	require.NoError(t, err)

	_, err = domain.Nodes[0].ForwarderRegistryClient.Unregister(ctx, nse2Reg)
	require.NoError(t, err)
}

func TestNSMGR_ShouldCleanAllClientAndEndpointGoroutines(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()

	// We have lazy initialization in some chain elements in both networkservice, registry chains. So registering an
	// endpoint and requesting it from client can result in new endless NSMgr goroutines.
	testNSEAndClient(ctx, t, domain, &registry.NetworkServiceEndpoint{
		Name:                "endpoint-init",
		NetworkServiceNames: []string{"service-init"},
	})

	// At this moment all possible endless NSMgr goroutines have been started. So we expect all newly created goroutines
	// to be canceled no later than some of these events:
	//   1. GRPC request context cancel
	//   2. NSC connection close
	//   3. NSE unregister
	testNSEAndClient(ctx, t, domain, &registry.NetworkServiceEndpoint{
		Name:                "endpoint-final",
		NetworkServiceNames: []string{"service-final"},
	})
}

func testNSEAndClient(
	ctx context.Context,
	t *testing.T,
	domain *sandbox.Domain,
	nseReg *registry.NetworkServiceEndpoint,
) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)
	require.NoError(t, err)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			NetworkService: nseReg.NetworkServiceNames[0],
		},
	})
	require.NoError(t, err)

	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	_, err = domain.Nodes[0].EndpointRegistryClient.Unregister(ctx, nseReg)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		stream, err := domain.Nodes[0].NSRegistryClient.Find(ctx, &registry.NetworkServiceQuery{
			NetworkService: new(registry.NetworkService),
		})
		require.NoError(t, err)
		return len(registry.ReadNetworkServiceList(stream)) == 0
	}, 100*time.Millisecond, 10*time.Millisecond)
}

type passThroughClient struct {
	networkService string
}

func newPassTroughClient(networkService string) *passThroughClient {
	return &passThroughClient{
		networkService: networkService,
	}
}

func (c *passThroughClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	request.Connection.NetworkService = c.networkService
	request.Connection.NetworkServiceEndpointName = ""
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *passThroughClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	conn.NetworkService = c.networkService
	conn.NetworkServiceEndpointName = ""
	return next.Client(ctx).Close(ctx, conn, opts...)
}

type counterServer struct {
	Requests, Closes int32
	requests         map[string]int32
	closes           map[string]int32
	mu               sync.Mutex
}

func (c *counterServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.AddInt32(&c.Requests, 1)
	if c.requests == nil {
		c.requests = make(map[string]int32)
	}
	c.requests[request.GetConnection().GetId()]++

	return next.Server(ctx).Request(ctx, request)
}

func (c *counterServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.AddInt32(&c.Closes, 1)
	if c.closes == nil {
		c.closes = make(map[string]int32)
	}
	c.closes[connection.GetId()]++

	return next.Server(ctx).Close(ctx, connection)
}

func (c *counterServer) UniqueRequests() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.requests == nil {
		return 0
	}
	return len(c.requests)
}

func (c *counterServer) UniqueCloses() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closes == nil {
		return 0
	}
	return len(c.closes)
}

type restartingEndpoint struct {
	startTime time.Time
}

func (c *restartingEndpoint) Request(ctx context.Context, req *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if time.Now().Before(c.startTime) {
		return nil, errors.New("endpoint is restarting")
	}
	return next.Server(ctx).Request(ctx, req)
}

func (c *restartingEndpoint) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	if time.Now().Before(c.startTime) {
		return nil, errors.New("endpoint is restarting")
	}
	return next.Server(ctx).Close(ctx, connection)
}

func newBusyEndpoint() networkservice.NetworkServiceServer {
	return injecterror.NewServer(errors.New("sorry, endpoint is busy, try again later"))
}

func TestNSMGR_LocalUsecaseNoURL(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix sockets are not supported under windows, skipping")
		return
	}
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		UseUnixSockets().
		SetContext(ctx).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{"my-service-remote"},
	}

	counter := &counterServer{}
	_, err := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service-remote",
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 5, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn2, err := nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn2)
	require.Equal(t, 5, len(conn2.Path.PathSegments))
	require.Equal(t, int32(2), atomic.LoadInt32(&counter.Requests))
	// Close.

	e, err := nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.NotNil(t, e)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Closes))
}
