// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023-2024 Cisco and/or its affiliates.
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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/passthrough"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/point2pointipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/strictipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	nodesCount = 7
)

type nsmgrSuite struct {
	suite.Suite

	domain           *sandbox.Domain
	nsRegistryClient registry.NetworkServiceRegistryClient
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

	s.nsRegistryClient = s.domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)
}

func (s *nsmgrSuite) Test_Remote_ParallelUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nsReg, err := s.nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.GetName())
	counter := new(count.Server)

	var unregisterWG sync.WaitGroup
	var nse *sandbox.EndpointEntry
	unregisterWG.Add(1)
	go func() {
		defer unregisterWG.Done()

		time.Sleep(time.Millisecond * 100)
		nse = s.domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)
	}()
	nsc := s.domain.Nodes[1].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.GetName())

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.Requests())
	require.Equal(t, 6, len(conn.GetPath().GetPathSegments()))

	// Simulate refresh from client.
	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 6, len(conn.GetPath().GetPathSegments()))
	require.Equal(t, 2, counter.Requests())

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, 1, counter.Closes())

	// Endpoint unregister
	unregisterWG.Wait()
	_, err = nse.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func (s *nsmgrSuite) Test_SelectsRestartingEndpointUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nsReg, err := s.nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.GetName())

	// 1. Start listen address and register endpoint
	netListener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer func() { _ = netListener.Close() }()
	nseReg.Url = "tcp://" + netListener.Addr().String()

	_, err = s.nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name:    nseReg.GetNetworkServiceNames()[0],
		Payload: payload.IP,
	})
	require.NoError(t, err)

	nseRegistryClient := registryclient.NewNetworkServiceEndpointRegistryClient(ctx,
		registryclient.WithClientURL(sandbox.CloneURL(s.domain.Nodes[0].NSMgr.URL)),
		registryclient.WithDialOptions(sandbox.DialOptions()...))

	nseReg, err = nseRegistryClient.Register(ctx, nseReg)
	require.NoError(t, err)

	// 2. Postpone endpoint start
	time.AfterFunc(time.Second, func() {
		serv := grpc.NewServer()
		endpoint.NewServer(ctx, sandbox.GenerateTestToken).Register(serv)
		_ = serv.Serve(netListener)
	})

	// 3. Create client and request endpoint
	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, defaultRequest(nsReg.GetName()))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 4, len(conn.GetPath().GetPathSegments()))

	require.NoError(t, ctx.Err())

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	// Endpoint unregister
	_, err = nseRegistryClient.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func (s *nsmgrSuite) Test_ReselectEndpointWhenNetSvcHasChanged() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nsReg, err := s.nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	deployNSE := func(name, ns string, labels map[string]string, ipNet *net.IPNet) {
		nseReg := defaultRegistryEndpoint(ns)
		nseReg.Name = name

		nseReg.NetworkServiceLabels = map[string]*registry.NetworkServiceLabels{
			ns: {
				Labels: labels,
			},
		}

		netListener, listenErr := net.Listen("tcp", "127.0.0.1:")
		s.Require().NoError(listenErr)

		nseReg.Url = "tcp://" + netListener.Addr().String()

		nseRegistryClient := registryclient.NewNetworkServiceEndpointRegistryClient(ctx,
			registryclient.WithClientURL(sandbox.CloneURL(s.domain.Nodes[0].NSMgr.URL)),
			registryclient.WithDialOptions(sandbox.DialOptions()...),
		)

		nseReg, err = nseRegistryClient.Register(ctx, nseReg)
		s.Require().NoError(err)

		go func() {
			<-ctx.Done()
			_ = netListener.Close()
		}()
		go func() {
			defer func() {
				_, _ = nseRegistryClient.Unregister(ctx, nseReg)
			}()

			serv := grpc.NewServer()
			endpoint.NewServer(ctx, sandbox.GenerateTestToken, endpoint.WithAdditionalFunctionality(
				strictipam.NewServer(point2pointipam.NewServer, ipNet),
			)).Register(serv)
			_ = serv.Serve(netListener)
		}()
	}

	_, ipNet1, _ := net.ParseCIDR("100.100.100.100/30")
	_, ipNet2, _ := net.ParseCIDR("200.200.200.200/30")
	deployNSE("nse-1", nsReg.GetName(), map[string]string{}, ipNet1)
	// 3. Create client and request endpoint
	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, defaultRequest(nsReg.GetName()))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 4, len(conn.GetPath().GetPathSegments()))
	require.Equal(t, "nse-1", conn.GetNetworkServiceEndpointName())
	require.NoError(t, ctx.Err())
	require.Equal(t, "100.100.100.101/32", conn.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	require.Equal(t, "100.100.100.100/32", conn.GetContext().GetIpContext().GetDstIpAddrs()[0])

	// update netsvc
	nsReg.Matches = append(nsReg.Matches, &registry.Match{
		Routes: []*registry.Destination{
			{
				DestinationSelector: map[string]string{
					"experimental": "true",
				},
			},
		},
	})
	nsReg, err = s.nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	// deploye nse-2 that matches with updated svc
	deployNSE("nse-2", nsReg.GetName(), map[string]string{
		"experimental": "true",
	}, ipNet2)
	// simulate idle
	time.Sleep(time.Second / 2)
	// in some moment nsc refresh connection
	conn, err = nsc.Request(ctx, &networkservice.NetworkServiceRequest{Connection: conn})
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 4, len(conn.GetPath().GetPathSegments()))
	require.Equal(t, "nse-2", conn.GetNetworkServiceEndpointName())
	require.NotEmpty(t, conn.GetContext().GetIpContext().GetSrcIpAddrs())
	require.NotEmpty(t, conn.GetContext().GetIpContext().GetDstIpAddrs())
	require.NoError(t, ctx.Err())
	require.Equal(t, "200.200.200.201/32", conn.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	require.Equal(t, "200.200.200.200/32", conn.GetContext().GetIpContext().GetDstIpAddrs()[0])

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
}

func (s *nsmgrSuite) Test_Remote_BusyEndpointsUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nsReg, err := s.nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	counter := new(count.Server)

	const nseCount = 3

	var wg sync.WaitGroup
	var nseRegs [nseCount + 1]*registry.NetworkServiceEndpoint
	var nses [nseCount + 1]*sandbox.EndpointEntry
	for i := 0; i < nseCount; i++ {
		wg.Add(1)
		go func(id int) {
			nseRegs[id] = defaultRegistryEndpoint(nsReg.GetName())
			nseRegs[id].Name += strconv.Itoa(id)

			nses[id] = s.domain.Nodes[1].NewEndpoint(ctx, nseRegs[id], sandbox.GenerateTestToken, injecterror.NewServer())
			wg.Done()
		}(i)
	}

	var unregisterWG sync.WaitGroup
	unregisterWG.Add(1)
	go func() {
		defer unregisterWG.Done()

		wg.Wait()
		time.Sleep(time.Second / 2)
		nseRegs[nseCount] = defaultRegistryEndpoint(nsReg.GetName())
		nseRegs[nseCount].Name += strconv.Itoa(3)

		nses[nseCount] = s.domain.Nodes[1].NewEndpoint(ctx, nseRegs[nseCount], sandbox.GenerateTestToken, counter)
	}()
	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.GetName())

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.Requests())
	require.Equal(t, 6, len(conn.GetPath().GetPathSegments()))

	// Simulate refresh from client
	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 2, counter.Requests())
	require.Equal(t, 6, len(conn.GetPath().GetPathSegments()))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, 1, counter.Closes())

	// Endpoint unregister
	unregisterWG.Wait()
	for i, nseReg := range nseRegs {
		_, err = nses[i].Unregister(ctx, nseReg)
		require.NoError(t, err)
	}
}

func (s *nsmgrSuite) Test_RemoteUsecase() {
	t := s.T()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nsReg, err := s.nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.GetName())
	counter := new(count.Server)

	nse := s.domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	nsc := s.domain.Nodes[1].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.GetName())

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.Requests())
	require.Equal(t, 6, len(conn.GetPath().GetPathSegments()))

	// Simulate refresh from client
	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 2, counter.Requests())
	require.Equal(t, 6, len(conn.GetPath().GetPathSegments()))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, 1, counter.Closes())

	// Endpoint unregister
	_, err = nse.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func (s *nsmgrSuite) Test_ConnectToDeadNSEUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nsReg, err := s.nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.GetName())
	counter := new(count.Server)

	nse := s.domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.GetName())

	reqCtx, reqCancel := context.WithTimeout(ctx, time.Second)
	defer reqCancel()

	conn, err := nsc.Request(reqCtx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.Requests())
	require.Equal(t, 4, len(conn.GetPath().GetPathSegments()))

	nse.Cancel()

	// Simulate refresh from client
	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	refreshCtx, refreshCancel := context.WithTimeout(ctx, time.Second)
	defer refreshCancel()
	_, err = nsc.Request(refreshCtx, refreshRequest)
	require.Error(t, err)
	require.NoError(t, ctx.Err())

	// Close
	closeCtx, closeCancel := context.WithTimeout(ctx, time.Second)
	defer closeCancel()
	_, _ = nsc.Close(closeCtx, conn)

	nseRegistryClient := registryclient.NewNetworkServiceEndpointRegistryClient(ctx,
		registryclient.WithClientURL(s.domain.Nodes[0].NSMgr.URL),
		registryclient.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	// Endpoint unregister
	_, err = nseRegistryClient.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func (s *nsmgrSuite) Test_LocalUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nsReg, err := s.nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.GetName())
	counter := new(count.Server)

	nse := s.domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.GetName())

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.Requests())
	require.Equal(t, 4, len(conn.GetPath().GetPathSegments()))

	// Simulate refresh from client
	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn2, err := nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn2)
	require.Equal(t, 4, len(conn2.GetPath().GetPathSegments()))
	require.Equal(t, 2, counter.Requests())

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, 1, counter.Closes())

	// Endpoint unregister
	_, err = nse.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func (s *nsmgrSuite) Test_PassThroughRemoteUsecase() {
	t := s.T()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	counterClose := new(count.Server)

	nsReg := linearNS(nodesCount)
	nsReg, err := s.nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	var nseRegs [nodesCount]*registry.NetworkServiceEndpoint
	var nses [nodesCount]*sandbox.EndpointEntry
	for i := range nseRegs {
		nseRegs[i], nses[i] = newPassThroughEndpoint(
			ctx,
			s.domain.Nodes[i],
			map[string]string{
				step: fmt.Sprintf("%v", i),
			},
			fmt.Sprintf("%v", i),
			nsReg.GetName(),
			i != nodesCount-1,
			counterClose,
		)
	}

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.GetName())

	// Request
	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length to first endpoint is 5
	// Path length from NSE client to other remote endpoint is 8
	require.Equal(t, 6*(nodesCount-1)+4, len(conn.GetPath().GetPathSegments()))
	for i := 0; i < len(nseRegs); i++ {
		require.Contains(t, nseRegs[i].GetName(), conn.GetPath().GetPathSegments()[i*6+3].GetName())
	}

	// Refresh
	for i := 0; i < 5; i++ {
		request.Connection = conn.Clone()

		conn, err = nsc.Request(ctx, request)
		require.NoError(t, err)
	}

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, nodesCount, counterClose.Closes())

	// Endpoint unregister
	for i, nseReg := range nseRegs {
		_, err = nses[i].Unregister(ctx, nseReg)
		require.NoError(t, err)
	}
}

func (s *nsmgrSuite) Test_PassThroughLocalUsecase() {
	t := s.T()
	const nsesCount = 7

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	counterClose := new(count.Server)

	nsReg, err := s.nsRegistryClient.Register(ctx, linearNS(nsesCount))
	require.NoError(t, err)

	var nseRegs [nsesCount]*registry.NetworkServiceEndpoint
	var nses [nsesCount]*sandbox.EndpointEntry
	for i := range nseRegs {
		nseRegs[i], nses[i] = newPassThroughEndpoint(
			ctx,
			s.domain.Nodes[0],
			map[string]string{
				step: fmt.Sprintf("%v", i),
			},
			fmt.Sprintf("%v", i),
			nsReg.GetName(),
			i != nsesCount-1,
			counterClose,
		)
	}

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.GetName())

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length to first endpoint is 5
	// Path length from NSE client to other local endpoint is 5
	require.Equal(t, 4*(nsesCount-1)+4, len(conn.GetPath().GetPathSegments()))
	for i := 0; i < len(nseRegs); i++ {
		require.Contains(t, nseRegs[i].GetName(), conn.GetPath().GetPathSegments()[(i+1)*4-1].GetName())
	}

	// Refresh
	for i := 0; i < 5; i++ {
		request.Connection = conn.Clone()

		conn, err = nsc.Request(ctx, request)
		require.NoError(t, err)
	}

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, nsesCount, counterClose.Closes())

	// Endpoint unregister
	for i, nseReg := range nseRegs {
		_, err = nses[i].Unregister(ctx, nseReg)
		require.NoError(t, err)
	}
}

func (s *nsmgrSuite) Test_PassThroughSameSourceSelector() {
	t := s.T()
	const nsesCount = 7

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	counterClose := new(count.Server)

	ns := linearNS(nsesCount)
	ns.Matches[len(ns.GetMatches())-1].Fallthrough = true
	ns.Matches = append(ns.Matches, &registry.Match{
		Routes: []*registry.Destination{
			{
				DestinationSelector: map[string]string{
					step: fmt.Sprintf("%v", 1),
				},
			},
		},
	})

	nsReg, err := s.nsRegistryClient.Register(ctx, ns)
	require.NoError(t, err)

	var nseRegs [nsesCount]*registry.NetworkServiceEndpoint
	var nses [nsesCount]*sandbox.EndpointEntry
	for i := range nseRegs {
		if i == 0 {
			continue
		}
		nseRegs[i], nses[i] = newPassThroughEndpoint(
			ctx,
			s.domain.Nodes[0],
			map[string]string{
				step: fmt.Sprintf("%v", i),
			},
			fmt.Sprintf("%v", i),
			nsReg.GetName(),
			i != nsesCount-1,
			counterClose,
		)
	}

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.GetName())

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length to first endpoint is 4
	// Path length from NSE client to other local endpoint is 4
	require.Equal(t, 4*(nsesCount-2)+4, len(conn.GetPath().GetPathSegments()))
	for i := 1; i < len(nseRegs); i++ {
		require.Contains(t, nseRegs[i].GetName(), conn.GetPath().GetPathSegments()[i*4-1].GetName())
	}

	// Refresh
	conn, err = nsc.Request(ctx, request)
	require.NoError(t, err)
	require.Equal(t, 4*(nsesCount-2)+4, len(conn.GetPath().GetPathSegments()))
	for i := 1; i < len(nseRegs); i++ {
		require.Contains(t, nseRegs[i].GetName(), conn.GetPath().GetPathSegments()[i*4-1].GetName())
	}

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, nsesCount-1, counterClose.Closes())

	// Endpoint unregister
	for i, nseReg := range nseRegs {
		if i == 0 {
			continue
		}
		_, err := nses[i].Unregister(ctx, nseReg)
		require.NoError(t, err)
	}
}

func (s *nsmgrSuite) Test_ShouldCleanAllClientAndEndpointGoroutines() {
	t := s.T()
	t.Cleanup(func() { goleak.VerifyNone(t, goleak.IgnoreCurrent()) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nsReg, err := s.nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	// At this moment all possible endless NSMgr goroutines have been started. So we expect all newly created goroutines
	// to be canceled no later than some of these events:
	//   1. GRPC request context cancel
	//   2. NSC connection close
	//   3. NSE unregister
	testNSEAndClient(ctx, t, s.domain, defaultRegistryEndpoint(nsReg.GetName()))
}

func (s *nsmgrSuite) Test_PassThroughLocalUsecaseMultiLabel() {
	t := s.T()
	t.Skip("unstable")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	counterClose := new(count.Server)

	nsReg, err := s.nsRegistryClient.Register(ctx, multiLabelNS())
	require.NoError(t, err)

	var labelAvalue, labelBvalue string
	var nseRegs [3 * 3]*registry.NetworkServiceEndpoint
	var nses [3 * 3]*sandbox.EndpointEntry
	for i := 0; i < 3; i++ {
		labelAvalue += "a"
		for j := 0; j < 3; j++ {
			labelBvalue += "b"
			nseRegs[i*3+j], nses[i*3+j] = newPassThroughEndpoint(
				ctx,
				s.domain.Nodes[0],
				map[string]string{
					labelA: labelAvalue,
					labelB: labelBvalue,
				},
				labelAvalue+labelBvalue,
				nsReg.GetName(),
				i != 2 || j != 2,
				counterClose,
			)
		}
		labelBvalue = ""
	}

	nsc := s.domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nsReg.GetName())

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length from NSE client to other local endpoint is 4
	expectedPath := []string{"ab", "aabb", "aaabbb"}
	require.Equal(t, 4*len(nsReg.GetMatches()), len(conn.GetPath().GetPathSegments()))
	for i := 0; i < len(expectedPath); i++ {
		require.Contains(t, conn.GetPath().GetPathSegments()[(i+1)*4-1].GetName(), expectedPath[i])
	}

	// Refresh
	conn, err = nsc.Request(ctx, request)
	require.NoError(t, err)

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, len(expectedPath), counterClose.Closes())

	// Endpoint unregister
	for i, nseReg := range nseRegs {
		_, err = nses[i].Unregister(ctx, nseReg)
		require.NoError(t, err)
	}
}

const (
	step   = "step"
	labelA = "label_a"
	labelB = "label_b"
)

func linearNS(hopsCount int) *registry.NetworkService {
	matches := make([]*registry.Match, 0)

	for i := 1; i < hopsCount; i++ {
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

	if hopsCount > 1 {
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
			connect.NewServer(
				client.NewClient(
					ctx,
					client.WithName(fmt.Sprintf("endpoint-client-%v", clientName)),
					client.WithAdditionalFunctionality(
						mechanismtranslation.NewClient(),
						passthrough.NewClient(labels),
						kernel.NewClient(),
					),
					client.WithDialOptions(sandbox.DialOptions()...),
					client.WithDialTimeout(sandbox.DialTimeout),
					client.WithoutRefresh(),
				),
			),
		),
	}
}

func newPassThroughEndpoint(
	ctx context.Context,
	node *sandbox.Node,
	labels map[string]string,
	name, nsRegName string,
	hasClientFunctionality bool,
	counter networkservice.NetworkServiceServer,
) (*registry.NetworkServiceEndpoint, *sandbox.EndpointEntry) {
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

	return nseReg, node.NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, additionalFunctionality...)
}
