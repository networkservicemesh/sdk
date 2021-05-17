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

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injectlabels"
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
		_, epErr := s.domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)
		require.NoError(t, epErr)
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

			_, epErr := s.domain.Nodes[1].NewEndpoint(ctx, nsesReg[id], sandbox.GenerateTestToken, injecterror.NewServer())
			require.NoError(t, epErr)
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

		_, epErr := s.domain.Nodes[1].NewEndpoint(ctx, nsesReg[3], sandbox.GenerateTestToken, counter)
		require.NoError(t, epErr)
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

	_, err = s.domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

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

	_, err = s.domain.Nodes[0].NewEndpoint(nseCtx, nseReg, sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

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

	nsReg, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)
	counter := &counterServer{}

	_, err = s.domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

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

	nsReg := linearNS(nodesCount)
	nsReg, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	var nsesReg [nodesCount]*registry.NetworkServiceEndpoint
	for i := 0; i < nodesCount; i++ {
		nsesReg[i] = &registry.NetworkServiceEndpoint{
			Name:                fmt.Sprintf("endpoint-%v", i),
			NetworkServiceNames: []string{nsReg.GetName()},
			NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
				nsReg.GetName(): {
					Labels: map[string]string{
						step: fmt.Sprintf("%v", i),
					},
				},
			},
		}

		var additionalFunctionality []networkservice.NetworkServiceServer
		if i != nodesCount-1 {
			additionalFunctionality = []networkservice.NetworkServiceServer{
				chain.NewNetworkServiceServer(
					clienturl.NewServer(s.domain.Nodes[i].NSMgr.URL),
					connect.NewServer(ctx,
						client.NewClientFactory(
							client.WithName(fmt.Sprintf("endpoint-client-%v", i)),
							client.WithAdditionalFunctionality(
								mechanismtranslation.NewClient(),
								injectlabels.NewClient(nsesReg[i].NetworkServiceLabels[nsReg.Name].Labels),
								kernel.NewClient(),
							),
						),
						connect.WithDialTimeout(sandbox.DialTimeout),
						connect.WithDialOptions(sandbox.DefaultDialOptions(sandbox.GenerateTestToken)...),
					),
				),
			}
		}

		_, err = s.domain.Nodes[i].NewEndpoint(ctx, nsesReg[i], sandbox.GenerateTestToken, additionalFunctionality...)
		require.NoError(t, err)
	}

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.GetName())

	// Request
	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length to first endpoint is 5
	// Path length from NSE client to other remote endpoint is 8
	require.Equal(t, 8*(nodesCount-1)+5, len(conn.Path.PathSegments))

	// Refresh
	conn, err = nsc.Request(ctx, request)
	require.NoError(t, err)

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
	const nsesCount = 5

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nsReg := linearNS(nsesCount)
	nsReg, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	var nsesReg [nsesCount]*registry.NetworkServiceEndpoint
	for i := 0; i < nsesCount; i++ {
		nsesReg[i] = &registry.NetworkServiceEndpoint{
			Name:                fmt.Sprintf("endpoint-%v", i),
			NetworkServiceNames: []string{nsReg.GetName()},
			NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
				nsReg.GetName(): {
					Labels: map[string]string{
						step: fmt.Sprintf("%v", i),
					},
				},
			},
		}

		var additionalFunctionality []networkservice.NetworkServiceServer
		if i != nsesCount-1 {
			additionalFunctionality = []networkservice.NetworkServiceServer{
				chain.NewNetworkServiceServer(
					clienturl.NewServer(s.domain.Nodes[0].NSMgr.URL),
					connect.NewServer(ctx,
						client.NewClientFactory(
							client.WithName(fmt.Sprintf("%v", i)),
							client.WithAdditionalFunctionality(
								mechanismtranslation.NewClient(),
								injectlabels.NewClient(nsesReg[i].NetworkServiceLabels[nsReg.Name].Labels),
								kernel.NewClient(),
							),
						),
						connect.WithDialTimeout(sandbox.DialTimeout),
						connect.WithDialOptions(sandbox.DefaultDialOptions(sandbox.GenerateTestToken)...),
					),
				),
			}
		}

		_, err = s.domain.Nodes[0].NewEndpoint(ctx, nsesReg[i], sandbox.GenerateTestToken, additionalFunctionality...)
		require.NoError(t, err)
	}

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.Name)

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length to first endpoint is 5
	// Path length from NSE client to other local endpoint is 5
	require.Equal(t, 5*(nsesCount-1)+5, len(conn.Path.PathSegments))

	// Refresh
	conn, err = nsc.Request(ctx, request)
	require.NoError(t, err)

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

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

	nsReg := multiLabelNS()
	nsReg, err := s.domain.Nodes[0].NSRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	var labelAvalue, labelBvalue string
	nsesReg := []*registry.NetworkServiceEndpoint{}
	for i := 0; i < 3; i++ {
		labelAvalue += "a"
		for j := 0; j < 3; j++ {
			labelBvalue += "b"

			nseReg := &registry.NetworkServiceEndpoint{
				Name:                fmt.Sprintf("endpoint-%v%v", i, j),
				NetworkServiceNames: []string{nsReg.GetName()},
				NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
					nsReg.GetName(): {
						Labels: map[string]string{
							labelA: labelAvalue,
							labelB: labelBvalue,
						},
					},
				},
			}

			var additionalFunctionality []networkservice.NetworkServiceServer
			if i != 2 || j != 2 {
				additionalFunctionality = []networkservice.NetworkServiceServer{
					chain.NewNetworkServiceServer(
						clienturl.NewServer(s.domain.Nodes[0].NSMgr.URL),
						connect.NewServer(ctx,
							client.NewClientFactory(
								client.WithName(fmt.Sprintf("endpoint-client-%v%v", i, j)),
								client.WithAdditionalFunctionality(
									mechanismtranslation.NewClient(),
									injectlabels.NewClient(nseReg.NetworkServiceLabels[nsReg.Name].Labels),
									kernel.NewClient(),
								),
							),
							connect.WithDialTimeout(sandbox.DialTimeout),
							connect.WithDialOptions(sandbox.DefaultDialOptions(sandbox.GenerateTestToken)...),
						),
					),
				}
			}

			_, err = s.domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, additionalFunctionality...)
			require.NoError(t, err)

			nsesReg = append(nsesReg, nseReg)
		}
		labelBvalue = ""
	}

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.Name)

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length from NSE client to other local endpoint is 5
	require.Equal(t, 5*len(nsReg.Matches), len(conn.Path.PathSegments))
	for i := 0; i < len(nsReg.Matches); i++ {
		require.Contains(t, nsesReg[i*4].Name, conn.Path.PathSegments[(i+1)*5-1].GetName())
	}

	// Refresh
	conn, err = nsc.Request(ctx, request)
	require.NoError(t, err)

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
