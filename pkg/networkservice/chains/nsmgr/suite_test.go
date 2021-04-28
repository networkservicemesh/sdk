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
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	nodesCount = 7
)

type nsmgrSuite struct {
	suite.Suite

	domain *sandbox.Domain
}

func TestNsmgr(t *testing.T) {
	suite.Run(t, new(nsmgrSuite))
}

func (s *nsmgrSuite) SetupSuite() {
	t := s.T()

	ctx, cancel := context.WithCancel(context.Background())

	// Call cleanup when tests complete
	t.Cleanup(func() {
		cancel()
		goleak.VerifyNone(s.T())
	})

	// Create default domain with nodesCount nodes, which will be enough for any test
	s.domain = sandbox.NewBuilder(t).
		SetNodesCount(nodesCount).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		SetContext(ctx).
		Build()

	_, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)
}

func (s *nsmgrSuite) Test_Remote_ParallelUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nseReg := defaultRegistryEndpoint()
	request := defaultRequest()
	counter := &counterServer{}

	var unregisterWG sync.WaitGroup
	unregisterWG.Add(1)
	go func() {
		defer unregisterWG.Done()

		time.Sleep(time.Millisecond * 100)
		_, err := s.domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)
		require.NoError(t, err)
	}()
	nsc := s.domain.Nodes[1].NewClient(ctx, sandbox.GenerateTestToken)

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

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Closes))

	// Endpoint unregister
	unregisterWG.Wait()
	_, err = s.domain.Nodes[0].EndpointRegistryClient.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func (s *nsmgrSuite) Test_SelectsRestartingEndpointUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nseReg := defaultRegistryEndpoint()
	request := defaultRequest()

	// 1. Start listen address and register endpoint
	netListener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer func() { _ = netListener.Close() }()
	nseReg.Url = "tcp://" + netListener.Addr().String()

	_, err = s.domain.Nodes[0].NSRegistryClient.Register(ctx, &registry.NetworkService{
		Name:    nseReg.NetworkServiceNames[0],
		Payload: payload.IP,
	})
	require.NoError(t, err)

	nseReg, err = s.domain.Nodes[0].EndpointRegistryClient.Register(ctx, nseReg)
	require.NoError(t, err)

	// 2. Postpone endpoint start
	time.AfterFunc(time.Second, func() {
		serv := grpc.NewServer()
		endpoint.NewServer(ctx, sandbox.GenerateTestToken).Register(serv)
		_ = serv.Serve(netListener)
	})

	// 3. Create client and request endpoint
	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 5, len(conn.Path.PathSegments))

	require.NoError(t, ctx.Err())

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	// Endpoint unregister
	_, err = s.domain.Nodes[0].EndpointRegistryClient.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func (s *nsmgrSuite) Test_Remote_BusyEndpointsUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	request := defaultRequest()
	counter := &counterServer{}

	var wg sync.WaitGroup
	var nsesReg [4]*registry.NetworkServiceEndpoint
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			nsesReg[id] = defaultRegistryEndpoint()
			nsesReg[id].Name += strconv.Itoa(id)

			_, err := s.domain.Nodes[1].NewEndpoint(ctx, nsesReg[id], sandbox.GenerateTestToken, injecterror.NewServer())
			require.NoError(t, err)
			wg.Done()
		}(i)
	}

	var unregisterWG sync.WaitGroup
	unregisterWG.Add(1)
	go func() {
		defer unregisterWG.Done()

		wg.Wait()
		time.Sleep(time.Second / 2)
		nsesReg[3] = defaultRegistryEndpoint()
		nsesReg[3].Name += strconv.Itoa(3)

		_, err := s.domain.Nodes[1].NewEndpoint(ctx, nsesReg[3], sandbox.GenerateTestToken, counter)
		require.NoError(t, err)
	}()
	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client
	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(2), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Closes))

	// Endpoint unregister
	unregisterWG.Wait()
	for i := 0; i < len(nsesReg); i++ {
		_, err = s.domain.Nodes[1].EndpointRegistryClient.Unregister(ctx, nsesReg[i])
		require.NoError(t, err)
	}
}

func (s *nsmgrSuite) Test_RemoteUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nseReg := defaultRegistryEndpoint()
	request := defaultRequest()
	counter := &counterServer{}

	_, err := s.domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

	nsc := s.domain.Nodes[1].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client
	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(2), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Closes))

	// Endpoint unregister
	_, err = s.domain.Nodes[0].EndpointRegistryClient.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func (s *nsmgrSuite) Test_ConnectToDeadNSEUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	nseCtx, killNse := context.WithCancel(ctx)
	defer cancel()

	nseReg := defaultRegistryEndpoint()
	request := defaultRequest()
	counter := &counterServer{}

	_, err := s.domain.Nodes[0].NewEndpoint(nseCtx, nseReg, sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

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

	// Close
	_, err = nsc.Close(ctx, conn)
	require.Error(t, err)

	// Endpoint unregister
	_, err = s.domain.Nodes[0].EndpointRegistryClient.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func (s *nsmgrSuite) Test_LocalUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nseReg := defaultRegistryEndpoint()
	request := defaultRequest()
	counter := &counterServer{}

	_, err := s.domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 5, len(conn.Path.PathSegments))

	// Simulate refresh from client
	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn2, err := nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn2)
	require.Equal(t, 5, len(conn2.Path.PathSegments))
	require.Equal(t, int32(2), atomic.LoadInt32(&counter.Requests))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Closes))

	// Endpoint unregister
	_, err = s.domain.Nodes[0].EndpointRegistryClient.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func (s *nsmgrSuite) Test_PassThroughRemoteUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var nsesReg [nodesCount]*registry.NetworkServiceEndpoint
	for i := 0; i < nodesCount; i++ {
		var additionalFunctionality []networkservice.NetworkServiceServer
		if i != 0 {
			// Passtrough to the node i-1
			additionalFunctionality = []networkservice.NetworkServiceServer{
				chain.NewNetworkServiceServer(
					clienturl.NewServer(s.domain.Nodes[i].NSMgr.URL),
					connect.NewServer(ctx,
						sandbox.NewCrossConnectClientFactory(sandbox.GenerateTestToken,
							newPassTroughClient(fmt.Sprintf("my-service-remote-%v", i-1)),
							kernel.NewClient()),
						connect.WithDialOptions(sandbox.DefaultDialOptions(sandbox.GenerateTestToken)...),
					),
				),
			}
		}

		nsReg := &registry.NetworkService{
			Name: fmt.Sprintf("my-service-remote-%v", i),
		}
		_, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, nsReg)
		require.NoError(t, err)

		nsesReg[i] = &registry.NetworkServiceEndpoint{
			Name:                fmt.Sprintf("endpoint-%v", i),
			NetworkServiceNames: []string{nsReg.Name},
		}
		_, err = s.domain.Nodes[i].NewEndpoint(ctx, nsesReg[i], sandbox.GenerateTestToken, additionalFunctionality...)
		require.NoError(t, err)
	}

	nsc := s.domain.Nodes[nodesCount-1].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest()
	request.Connection.NetworkService = fmt.Sprintf("my-service-remote-%v", nodesCount-1)

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length to first endpoint is 5
	// Path length from NSE client to other remote endpoint is 8
	require.Equal(t, 8*(nodesCount-1)+5, len(conn.Path.PathSegments))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	// Endpoint unregister
	for i := 0; i < len(nsesReg); i++ {
		_, err = s.domain.Nodes[i].EndpointRegistryClient.Unregister(ctx, nsesReg[i])
		require.NoError(t, err)
	}
}

func (s *nsmgrSuite) Test_PassThroughLocalUsecase() {
	t := s.T()
	const nsesCount = 7

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var nsesReg [nsesCount]*registry.NetworkServiceEndpoint
	for i := 0; i < nsesCount; i++ {
		var additionalFunctionality []networkservice.NetworkServiceServer
		if i != 0 {
			additionalFunctionality = []networkservice.NetworkServiceServer{
				chain.NewNetworkServiceServer(
					clienturl.NewServer(s.domain.Nodes[0].NSMgr.URL),
					connect.NewServer(ctx,
						sandbox.NewCrossConnectClientFactory(sandbox.GenerateTestToken,
							newPassTroughClient(fmt.Sprintf("my-service-remote-%v", i-1)),
							kernel.NewClient()),
						connect.WithDialOptions(sandbox.DefaultDialOptions(sandbox.GenerateTestToken)...),
					),
				),
			}
		}

		nsReg := &registry.NetworkService{
			Name: fmt.Sprintf("my-service-remote-%v", i),
		}
		_, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, nsReg)
		require.NoError(t, err)

		nsesReg[i] = &registry.NetworkServiceEndpoint{
			Name:                fmt.Sprintf("endpoint-%v", i),
			NetworkServiceNames: []string{nsReg.Name},
		}
		_, err = s.domain.Nodes[0].NewEndpoint(ctx, nsesReg[i], sandbox.GenerateTestToken, additionalFunctionality...)
		require.NoError(t, err)
	}

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest()
	request.Connection.NetworkService = fmt.Sprintf("my-service-remote-%v", nsesCount-1)

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length to first endpoint is 5
	// Path length from NSE client to other local endpoint is 5
	require.Equal(t, 5*(nsesCount-1)+5, len(conn.Path.PathSegments))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	// Endpoint unregister
	for i := 0; i < len(nsesReg); i++ {
		_, err = s.domain.Nodes[0].EndpointRegistryClient.Unregister(ctx, nsesReg[i])
		require.NoError(t, err)
	}
}

func (s *nsmgrSuite) Test_ShouldCleanAllClientAndEndpointGoroutines() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	// At this moment all possible endless NSMgr goroutines have been started. So we expect all newly created goroutines
	// to be canceled no later than some of these events:
	//   1. GRPC request context cancel
	//   2. NSC connection close
	//   3. NSE unregister
	testNSEAndClient(ctx, t, s.domain, defaultRegistryEndpoint())
}
