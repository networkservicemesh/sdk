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
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/opentracing"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func TestNSMGR_RemoteUsecase_Parallel(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()
	defer domain.Cleanup()

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
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()
	defer domain.Cleanup()

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

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "nse-1",
		NetworkServiceNames: []string{"ns-1"},
	}

	_, err := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, &restartingEndpoint{startTime: time.Now().Add(time.Second * 2)})
	require.NoError(t, err)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 5, len(conn.Path.PathSegments))

	require.NoError(t, ctx.Err())
}

func TestNSMGR_RemoteUsecase_BusyEndpoints(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()
	defer domain.Cleanup()

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
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()
	defer domain.Cleanup()

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
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetContext(ctx).
		SetRegistryProxySupplier(nil).
		Build()
	defer domain.Cleanup()

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
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetContext(ctx).
		SetRegistryProxySupplier(nil).
		Build()
	defer domain.Cleanup()

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
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	const nodesCount = 7
	domain := sandbox.NewBuilder(t).
		SetNodesCount(nodesCount).
		SetContext(ctx).
		SetRegistryProxySupplier(nil).
		Build()
	defer domain.Cleanup()

	for i := 0; i < nodesCount; i++ {
		var additionalFunctionality []networkservice.NetworkServiceServer
		if i != 0 {
			k := i
			// Passtrough to the node i-1
			additionalFunctionality = []networkservice.NetworkServiceServer{
				chain.NewNetworkServiceServer(
					clienturl.NewServer(domain.Nodes[i].NSMgr.URL),
					connect.NewServer(ctx,
						sandbox.NewCrossConnectClientFactory(sandbox.GenerateTestToken,
							newPassTroughClient(
								fmt.Sprintf("my-service-remote-%v", k-1),
								fmt.Sprintf("endpoint-%v", k-1)),
							kernel.NewClient()),
						append(opentracing.WithTracingDial(), grpc.WithBlock(), grpc.WithInsecure())...,
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
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	const nsesCount = 7
	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetContext(ctx).
		SetRegistryProxySupplier(nil).
		Build()
	defer domain.Cleanup()

	for i := 0; i < nsesCount; i++ {
		var additionalFunctionality []networkservice.NetworkServiceServer
		if i != 0 {
			k := i
			additionalFunctionality = []networkservice.NetworkServiceServer{
				chain.NewNetworkServiceServer(
					clienturl.NewServer(domain.Nodes[0].NSMgr.URL),
					connect.NewServer(ctx,
						sandbox.NewCrossConnectClientFactory(sandbox.GenerateTestToken,
							newPassTroughClient(
								fmt.Sprintf("my-service-remote-%v", k-1),
								fmt.Sprintf("endpoint-%v", k-1)),
							kernel.NewClient()),
						append(opentracing.WithTracingDial(), grpc.WithBlock(), grpc.WithInsecure())...,
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

func TestNSMGR_ShouldCleanAllClientAndEndpointGoroutines(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()
	defer domain.Cleanup()

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

	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

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
	networkService             string
	networkServiceEndpointName string
}

func newPassTroughClient(networkService, networkServiceEndpointName string) *passThroughClient {
	return &passThroughClient{
		networkService:             networkService,
		networkServiceEndpointName: networkServiceEndpointName,
	}
}

func (c *passThroughClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	request.Connection.NetworkService = c.networkService
	request.Connection.NetworkServiceEndpointName = c.networkServiceEndpointName
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *passThroughClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	conn.NetworkService = c.networkService
	conn.NetworkServiceEndpointName = c.networkServiceEndpointName
	return next.Client(ctx).Close(ctx, conn, opts...)
}

type counterServer struct {
	Requests, Closes int32
}

func (c *counterServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	atomic.AddInt32(&c.Requests, 1)
	return next.Server(ctx).Request(ctx, request)
}

func (c *counterServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	atomic.AddInt32(&c.Closes, 1)
	return next.Server(ctx).Close(ctx, connection)
}

type restartingEndpoint struct {
	startTime time.Time
}

func (c *restartingEndpoint) Request(ctx context.Context, req *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if time.Now().Before(c.startTime) {
		return nil, errors.New("endpoint is restarting")
	}
	return next.Client(ctx).Request(ctx, req)
}

func (c *restartingEndpoint) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	if time.Now().Before(c.startTime) {
		return nil, errors.New("endpoint is restarting")
	}
	return next.Client(ctx).Close(ctx, connection)
}

type busyEndpoint struct{}

func (c *busyEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	return nil, errors.New("sorry, endpoint is busy, try again later")
}

func (c *busyEndpoint) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	return nil, errors.New("sorry, endpoint is busy, try again later")
}
func newBusyEndpoint() networkservice.NetworkServiceServer {
	return new(busyEndpoint)
}
