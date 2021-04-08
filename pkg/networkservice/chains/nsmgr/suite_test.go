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
	"net/url"
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

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/replacelabels"
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
	s.domain = sandbox.NewBuilder(ctx, t).
		SetNodesCount(nodesCount).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		Build()
}

func (s *nsmgrSuite) Test_Remote_ParallelUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nsReg, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)
	counter := &counterServer{}

	var unregisterWG sync.WaitGroup
	unregisterWG.Add(1)
	go func() {
		defer unregisterWG.Done()

		time.Sleep(time.Millisecond * 100)
		s.domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)
	}()
	nsc := s.domain.Nodes[1].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.Name)

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

	nsReg, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

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

	conn, err := nsc.Request(ctx, defaultRequest(nsReg.Name))
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

	nsReg, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	counter := &counterServer{}

	var wg sync.WaitGroup
	var nsesReg [4]*registry.NetworkServiceEndpoint
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			nsesReg[id] = defaultRegistryEndpoint(nsReg.Name)
			nsesReg[id].Name += strconv.Itoa(id)

			s.domain.Nodes[1].NewEndpoint(ctx, nsesReg[id], sandbox.GenerateTestToken, injecterror.NewServer())
			wg.Done()
		}(i)
	}

	var unregisterWG sync.WaitGroup
	unregisterWG.Add(1)
	go func() {
		defer unregisterWG.Done()

		wg.Wait()
		time.Sleep(time.Second / 2)
		nsesReg[3] = defaultRegistryEndpoint(nsReg.Name)
		nsesReg[3].Name += strconv.Itoa(3)

		s.domain.Nodes[1].NewEndpoint(ctx, nsesReg[3], sandbox.GenerateTestToken, counter)
	}()
	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.Name)

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

	nsReg, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)
	counter := &counterServer{}

	s.domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	nsc := s.domain.Nodes[1].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.Name)

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

	nsReg, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)
	counter := &counterServer{}

	s.domain.Nodes[0].NewEndpoint(nseCtx, nseReg, sandbox.GenerateTestToken, counter)

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.Name)

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

	// Endpoint unregister
	_, err = s.domain.Nodes[0].EndpointRegistryClient.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func (s *nsmgrSuite) Test_LocalUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nsReg, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)
	counter := &counterServer{}

	s.domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.Name)

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

	counterClose := &counterServer{}

	nsReg := linearNS(nodesCount)
	nsReg, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	var nsesReg []*registry.NetworkServiceEndpoint
	for i := 0; i < nodesCount; i++ {
		labels := map[string]string{
			step: fmt.Sprintf("%v", i),
		}
		nsesReg = append(nsesReg, newPassThroughEndpoint(ctx, s.domain.Nodes[i], labels, fmt.Sprintf("%v", i), nsReg.Name, i != nodesCount-1, counterClose))
	}

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.Name)

	// Request
	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length to first endpoint is 5
	// Path length from NSE client to other remote endpoint is 8
	require.Equal(t, 8*(nodesCount-1)+5, len(conn.Path.PathSegments))
	for i := 0; i < len(nsesReg); i++ {
		require.Contains(t, nsesReg[i].Name, conn.Path.PathSegments[i*8+4].Name)
	}

	// Refresh
	conn, err = nsc.Request(ctx, request)
	require.NoError(t, err)

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, int32(nodesCount), atomic.LoadInt32(&counterClose.Closes))

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

	counterClose := &counterServer{}

	nsReg := linearNS(nsesCount)
	nsReg, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	var nsesReg []*registry.NetworkServiceEndpoint
	for i := 0; i < nsesCount; i++ {
		labels := map[string]string{
			step: fmt.Sprintf("%v", i),
		}

		nsesReg = append(nsesReg, newPassThroughEndpoint(ctx, s.domain.Nodes[0], labels, fmt.Sprintf("%v", i), nsReg.Name, i != nsesCount-1, counterClose))
	}

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.Name)

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length to first endpoint is 5
	// Path length from NSE client to other local endpoint is 5
	require.Equal(t, 5*(nsesCount-1)+5, len(conn.Path.PathSegments))
	for i := 0; i < len(nsesReg); i++ {
		require.Contains(t, nsesReg[i].Name, conn.Path.PathSegments[(i+1)*5-1].Name)
	}

	// Refresh
	conn, err = nsc.Request(ctx, request)
	require.NoError(t, err)

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, int32(nsesCount), atomic.LoadInt32(&counterClose.Closes))

	// Endpoint unregister
	for i := 0; i < len(nsesReg); i++ {
		_, err = s.domain.Nodes[0].EndpointRegistryClient.Unregister(ctx, nsesReg[i])
		require.NoError(t, err)
	}
}

func (s *nsmgrSuite) Test_PassThroughLocalUsecaseMultiLabel() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	counterClose := &counterServer{}

	nsReg := multiLabelNS()
	nsReg, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	var labelAvalue, labelBvalue string
	nsesReg := []*registry.NetworkServiceEndpoint{}
	for i := 0; i < 3; i++ {
		labelAvalue += "a"
		for j := 0; j < 3; j++ {
			labelBvalue += "b"

			labels := map[string]string{
				labelA: labelAvalue,
				labelB: labelBvalue,
			}

			nsesReg = append(nsesReg, newPassThroughEndpoint(ctx, s.domain.Nodes[0], labels, labelAvalue+labelBvalue, nsReg.Name, i != 2 || j != 2, counterClose))
		}
		labelBvalue = ""
	}

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.Name)

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length from NSE client to other local endpoint is 5
	expectedPath := []string{"ab", "aabb", "aaabbb"}
	require.Equal(t, 5*len(nsReg.Matches), len(conn.Path.PathSegments))
	for i := 0; i < len(expectedPath); i++ {
		require.Contains(t, conn.Path.PathSegments[(i+1)*5-1].Name, expectedPath[i])
	}

	// Refresh
	conn, err = nsc.Request(ctx, request)
	require.NoError(t, err)

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, int32(len(expectedPath)), atomic.LoadInt32(&counterClose.Closes))

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

	nsReg, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	// At this moment all possible endless NSMgr goroutines have been started. So we expect all newly created goroutines
	// to be canceled no later than some of these events:
	//   1. GRPC request context cancel
	//   2. NSC connection close
	//   3. NSE unregister
	testNSEAndClient(ctx, t, s.domain, defaultRegistryEndpoint(nsReg.Name))
}

const (
	step   = "step"
	labelA = "label_a"
	labelB = "label_b"
)

func linearNS(count int) *registry.NetworkService {
	matches := make([]*registry.Match, 0)

	for i := 1; i < count; i++ {
		match := &registry.Match{
			SourceSelector: map[string]string{
				step: fmt.Sprintf("%v", i-1),
			},
			Routes: []*registry.Destination{
				{
					DestinationSelector: map[string]string{
						step: fmt.Sprintf("%v", i),
					},
				},
			},
		}

		matches = append(matches, match)
	}

	if count > 1 {
		// match with empty source selector must be the last
		match := &registry.Match{
			Routes: []*registry.Destination{
				{
					DestinationSelector: map[string]string{
						step: fmt.Sprintf("%v", 0),
					},
				},
			},
		}
		matches = append(matches, match)
	}

	return &registry.NetworkService{
		Name:    "test-network-service-linear",
		Matches: matches,
	}
}

func multiLabelNS() *registry.NetworkService {
	return &registry.NetworkService{
		Name: "test-network-service",
		Matches: []*registry.Match{
			{
				SourceSelector: map[string]string{
					labelA: "a",
					labelB: "b",
				},
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{
							labelA: "aa",
							labelB: "bb",
						},
					},
				},
			},
			{
				SourceSelector: map[string]string{
					labelA: "aa",
					labelB: "bb",
				},
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{
							labelA: "aaa",
							labelB: "bbb",
						},
					},
				},
			},
			{
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{
							labelA: "a",
							labelB: "b",
						},
					},
				},
			},
		},
	}
}

func additionalFunctionalityChain(ctx context.Context, clientURL *url.URL, clientName string, labels map[string]string) []networkservice.NetworkServiceServer {
	return []networkservice.NetworkServiceServer{
		chain.NewNetworkServiceServer(
			clienturl.NewServer(clientURL),
			connect.NewServer(ctx,
				client.NewClientFactory(
					client.WithName(fmt.Sprintf("endpoint-client-%v", clientName)),
					client.WithAdditionalFunctionality(
						mechanismtranslation.NewClient(),
						replacelabels.NewClient(labels),
						kernel.NewClient(),
					),
				),
				connect.WithDialTimeout(sandbox.DialTimeout),
				connect.WithDialOptions(sandbox.DefaultDialOptions(sandbox.GenerateTestToken)...),
			),
		),
	}
}

func newPassThroughEndpoint(ctx context.Context, node *sandbox.Node, labels map[string]string, name, nsRegName string, hasClientFunctionality bool, counter networkservice.NetworkServiceServer) *registry.NetworkServiceEndpoint {
	nseReg := &registry.NetworkServiceEndpoint{
		Name:                fmt.Sprintf("endpoint-%v", name),
		NetworkServiceNames: []string{nsRegName},
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			nsRegName: {
				Labels: labels,
			},
		},
	}

	var additionalFunctionality []networkservice.NetworkServiceServer
	if hasClientFunctionality {
		additionalFunctionality = additionalFunctionalityChain(ctx, node.NSMgr.URL, name, labels)
	}

	if counter != nil {
		additionalFunctionality = append(additionalFunctionality, counter)
	}

	node.NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, additionalFunctionality...)

	return nseReg
}
