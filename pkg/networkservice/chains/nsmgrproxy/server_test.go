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

package nsmgrproxy_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	kernelmech "github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

// TestNSMGR_InterdomainUseCase covers simple interdomain scenario:
//
//	nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy2 -> nsmgr2 ->forwarder2 -> nsmgr2 -> nse
func TestNSMGR_InterdomainUseCase(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var ctx, cancel = context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var domains = sandbox.NewInterdomainBuilder(ctx, t).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(1).SetName("cluster1")
		}).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(1).SetName("cluster2")
		}).
		Build()

	nsRegistryClient := domains[1].NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := &registry.NetworkService{
		Name: "my-service-interdomain",
	}

	_, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{nsReg.Name},
	}

	domains[1].Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, checkrequest.NewServer(t, func(t *testing.T, nsr *networkservice.NetworkServiceRequest) {
		require.False(t, interdomain.Is(nsr.GetConnection().GetNetworkService()))
	}))

	nsc := domains[0].Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: fmt.Sprint(nsReg.Name, "@", domains[1].Name),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
}

// TestNSMGR_InterdomainUseCase covers simple interdomain scenario:
//
// request1: nsc -> nsmgr1 ->  forwarder1  -> nsmgr-proxy1 -> nsmg-proxy2 -> nsmgr2 ->forwarder2 -> final-endpoint via cluster2
// request2: nsc -> nsmgr1 ->  forwarder1  -> nsmgr-proxy1 -> nsmg-proxy2 -> nsmgr2 ->forwarder2 -> final-endpoint via floating registry
func Test_NSEMovedFromInterdomainToFloatingUseCase(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var domains = sandbox.NewInterdomainBuilder(ctx, t).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(1).SetName("cluster1")
		}).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(1).SetName("cluster2")
		}).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(0).SetRegistryProxySupplier(nil).SetNSMgrProxySupplier(nil).SetName("floating")
		}).
		Build()

	nsRegistryClient := domains[1].NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg1 := &registry.NetworkService{
		Name: "my-service-interdomain",
	}

	var err error

	nsReg1, err = nsRegistryClient.Register(ctx, nsReg1)
	require.NoError(t, err)

	nsReg2 := &registry.NetworkService{
		Name: "my-service-interdomain@" + domains[2].Name,
	}

	nsReg2, err = nsRegistryClient.Register(ctx, nsReg2)
	require.NoError(t, err)

	nseReg1 := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{nsReg1.Name},
	}

	domains[1].Nodes[0].NewEndpoint(ctx, nseReg1, sandbox.GenerateTestToken)

	nseReg2 := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint@" + domains[2].Name,
		NetworkServiceNames: []string{nsReg1.Name},
	}

	domains[1].Nodes[0].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken)

	stream, err := adapters.NetworkServiceEndpointServerToClient(domains[2].Registry.NetworkServiceEndpointRegistryServer()).Find(context.Background(), &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
		Name: nseReg1.Name,
	}})
	require.NoError(t, err)
	require.Len(t, registry.ReadNetworkServiceEndpointList(stream), 1)

	stream, err = adapters.NetworkServiceEndpointServerToClient(domains[2].Registry.NetworkServiceEndpointRegistryServer()).Find(context.Background(), &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
		Name: nseReg1.Name,
	}})
	require.NoError(t, err)
	require.Len(t, registry.ReadNetworkServiceEndpointList(stream), 1)

	nsc := domains[0].Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	var finalNSE = map[string]string{
		fmt.Sprint(nsReg1.Name, "@", domains[2].Name): nseReg1.GetName(),
		nsReg2.GetName(): nseReg2.GetName(),
	}

	for nsName := range finalNSE {
		request := &networkservice.NetworkServiceRequest{
			MechanismPreferences: []*networkservice.Mechanism{
				{Cls: cls.LOCAL, Type: kernel.MECHANISM},
			},
			Connection: &networkservice.Connection{
				Id:             "1",
				NetworkService: nsName,
				Context:        &networkservice.ConnectionContext{},
			},
		}

		conn, err := nsc.Request(ctx, request)
		require.NoError(t, err)
		require.NotNil(t, conn)

		require.Equal(t, 8, len(conn.Path.PathSegments))

		require.Equal(t, finalNSE[nsName], conn.GetPath().GetPathSegments()[7].GetName())

		// Simulate refresh from client.

		refreshRequest := request.Clone()
		refreshRequest.Connection = conn.Clone()

		conn, err = nsc.Request(ctx, refreshRequest)
		require.NoError(t, err)
		require.NotNil(t, conn)
		require.Equal(t, 8, len(conn.Path.PathSegments))
		require.Equal(t, finalNSE[nsName], conn.GetPath().GetPathSegments()[7].GetName())

		// Close
		_, err = nsc.Close(ctx, conn)
		require.NoError(t, err)
	}
}

// TestNSMGR_Interdomain_TwoNodesNSEs covers scenarion with connection from the one client to two endpoints from diffrenret clusters.
//
//	nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy2 -> nsmgr2 ->forwarder2 -> nsmgr2 -> nse2
//	nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy3 -> nsmgr3 ->forwarder3 -> nsmgr3 -> nse3
func TestNSMGR_Interdomain_TwoNodesNSEs(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var domains = sandbox.NewInterdomainBuilder(ctx, t).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(1).SetName("cluster1")
		}).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(1).SetName("cluster2")
		}).
		Build()

	cluster1 := domains[0]
	cluster2 := domains[1]

	nsRegistryClient := cluster2.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	_, err := nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-service-interdomain-1",
	})
	require.NoError(t, err)

	_, err = nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-service-interdomain-2",
	})
	require.NoError(t, err)

	nseReg1 := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint-1",
		NetworkServiceNames: []string{"my-service-interdomain-1"},
	}
	cluster2.Nodes[0].NewEndpoint(ctx, nseReg1, sandbox.GenerateTestToken)

	nseReg2 := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint-2",
		NetworkServiceNames: []string{"my-service-interdomain-2"},
	}
	cluster2.Nodes[0].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken)

	nsc := cluster1.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: fmt.Sprint("my-service-interdomain-1", "@", cluster2.Name),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))

	request = &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "2",
			NetworkService: fmt.Sprint("my-service-interdomain-2", "@", cluster2.Name),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err = nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest = request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))
}

// TestNSMGR_FloatingInterdomainUseCase covers simple interdomain scenario with resolving endpoint from floating registry:
//
//	nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy2 -> nsmgr2 ->forwarder2 -> nsmgr2 -> nse3
func TestNSMGR_FloatingInterdomainUseCase(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Hour*5)
	defer cancel()

	var domains = sandbox.NewInterdomainBuilder(ctx, t).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(1).SetName("cluster1")
		}).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(1).SetName("cluster2")
		}).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(0).SetNSMgrProxySupplier(nil).SetRegistryProxySupplier(nil).SetName("floating")
		}).
		Build()

	cluster1 := domains[0]
	cluster2 := domains[1]
	floating := domains[2]

	nsRegistryClient := cluster2.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := &registry.NetworkService{
		Name: "my-service-interdomain@" + floating.Name,
	}

	_, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint@" + floating.Name,
		NetworkServiceNames: []string{"my-service-interdomain"},
	}

	cluster2.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	c := adapters.NetworkServiceEndpointServerToClient(floating.Registry.NetworkServiceEndpointRegistryServer())

	s, err := c.Find(ctx, &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
		Name: "final-endpoint",
	}})

	require.NoError(t, err)

	list := registry.ReadNetworkServiceEndpointList(s)

	require.Len(t, list, 1)

	nsc := cluster1.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: fmt.Sprint(nsReg.Name),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
}

// TestNSMGR_FloatingInterdomainUseCase_FloatingNetworkServiceNameRegistration covers simple interdomain scenario with resolving endpoint from floating registry:
//
// registration: {"name": "nse@floating.domain", "networkservice_names": ["my-service-interdomain@floating.domain"]}
//
//	nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy2 -> nsmgr2 ->forwarder2 -> nsmgr2 -> nse3
func TestNSMGR_FloatingInterdomainUseCase_FloatingNetworkServiceNameRegistration(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var domains = sandbox.NewInterdomainBuilder(ctx, t).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(1).SetName("cluster1")
		}).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(1).SetName("cluster2")
		}).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(0).SetNSMgrProxySupplier(nil).SetRegistryProxySupplier(nil).SetName("floating")
		}).
		Build()

	cluster1 := domains[0]
	cluster2 := domains[1]
	floating := domains[2]

	nsRegistryClient := cluster2.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := &registry.NetworkService{
		Name: "my-service-interdomain@" + floating.Name,
	}

	_, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint@" + floating.Name,
		NetworkServiceNames: []string{"my-service-interdomain@" + floating.Name},
	}

	cluster2.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	nsc := cluster1.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: fmt.Sprint(nsReg.Name),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
}

// TestNSMGR_FloatingInterdomain_FourClusters covers scenarion with connection from the one client to two endpoints
// from diffrenret clusters using floating registry for resolving endpoints.
//
//	nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy2 -> nsmgr2 ->forwarder2 -> nsmgr2 -> nse2
//	nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy3 -> nsmgr3 ->forwarder3 -> nsmgr3 -> nse3
func TestNSMGR_FloatingInterdomain_FourClusters(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// setup clusters

	var domains = sandbox.NewInterdomainBuilder(ctx, t).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(1).SetName("cluster1")
		}).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(1).SetName("cluster2")
		}).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(1).SetName("cluster3")
		}).
		BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.SetNodesCount(0).SetNSMgrProxySupplier(nil).SetRegistryProxySupplier(nil).SetName("floating")
		}).
		Build()

	cluster1 := domains[0]
	cluster2 := domains[1]
	cluster3 := domains[2]
	floating := domains[3]

	// register first ednpoint

	nsRegistryClient := cluster2.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg1 := &registry.NetworkService{
		Name: "my-service-interdomain-1@" + floating.Name,
	}

	_, err := nsRegistryClient.Register(ctx, nsReg1)
	require.NoError(t, err)

	nseReg1 := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint-1@" + floating.Name,
		NetworkServiceNames: []string{"my-service-interdomain-1"},
	}

	cluster2.Nodes[0].NewEndpoint(ctx, nseReg1, sandbox.GenerateTestToken)

	nsReg2 := &registry.NetworkService{
		Name: "my-service-interdomain-1@" + floating.Name,
	}

	// register second ednpoint

	nsRegistryClient = cluster3.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	_, err = nsRegistryClient.Register(ctx, nsReg2)
	require.NoError(t, err)

	nseReg2 := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint-2@" + floating.Name,
		NetworkServiceNames: []string{"my-service-interdomain-2"},
	}

	cluster3.Nodes[0].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken)

	// connect to first endpoint from cluster2

	nsc := cluster1.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: fmt.Sprint(nsReg1.Name),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// connect to second endpoint from cluster3
	request = &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "2",
			NetworkService: fmt.Sprint(nsReg2.Name),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err = nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest = request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))
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
	return next.Client(ctx).Close(ctx, conn, opts...)
}

// Test_Interdomain_PassThroughUsecase covers scenario when we have 5 clusters.
// Each cluster contains NSE with name endpoint-${cluster-num}.
// Each endpoint request endpoint from the previous cluster (exclude 1st).
//
//
// nsc -> nsmgr4 ->  forwarder4 -> nsmgr4 -> nsmgr-proxy4 -> nsmgr-proxy3 -> nsmgr3 ->forwarder3 -> nsmgr3 -> nse3 ->
// nse3 -> nsmgr3 ->  forwarder3 -> nsmgr3 -> nsmgr-proxy3 -> nsmgr-proxy2 -> nsmgr2 ->forwarder2-> nsmgr2 -> nse2 ->
// nse2 -> nsmgr2 ->  forwarder2 -> nsmgr2 -> nsmgr-proxy2 -> nsmgr-proxy1 -> nsmgr1 -> forwarder1 -> nsmgr1 -> nse1 ->
// nse1 -> nsmgr1 ->  forwarder1 -> nsmg1 -> nsmgr-proxy1 -> nsmgr-proxy0 -> nsmgr0 -> forwarder0 -> nsmgr0 -> nse0

func Test_Interdomain_PassThroughUsecase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	const clusterCount = 4

	var builder = sandbox.NewInterdomainBuilder(ctx, t)

	for i := 0; i < clusterCount; i++ {
		var index = i
		builder.BuildDomain(func(b *sandbox.Builder) *sandbox.Builder {
			return b.
				SetNodesCount(1).
				SetName("cluster" + fmt.Sprint(index))
		})
	}

	var clusters = builder.Build()
	require.Equal(t, clusterCount, len(clusters))
	for i := 0; i < clusterCount; i++ {
		var additionalFunctionality []networkservice.NetworkServiceServer
		if i != 0 {
			// Passtrough to the node i-1
			additionalFunctionality = []networkservice.NetworkServiceServer{
				chain.NewNetworkServiceServer(
					clienturl.NewServer(clusters[i].Nodes[0].NSMgr.URL),
					connect.NewServer(
						client.NewClient(
							ctx,
							client.WithAdditionalFunctionality(
								newPassTroughClient(fmt.Sprintf("my-service-remote-%v@cluster%v", i-1, i-1)),
								kernelmech.NewClient(),
							),
							client.WithDialTimeout(sandbox.DialTimeout),
							client.WithDialOptions(sandbox.DialOptions()...),
							client.WithoutRefresh(),
						),
					),
				),
			}
		}

		nsRegistryClient := clusters[i].NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

		nsReg, err := nsRegistryClient.Register(ctx, &registry.NetworkService{
			Name: fmt.Sprintf("my-service-remote-%v", i),
		})
		require.NoError(t, err)

		nsesReg := &registry.NetworkServiceEndpoint{
			Name:                fmt.Sprintf("endpoint-%v", i),
			NetworkServiceNames: []string{nsReg.Name},
		}
		clusters[i].Nodes[0].NewEndpoint(ctx, nsesReg, sandbox.GenerateTestToken, additionalFunctionality...)
	}

	nsc := clusters[clusterCount-1].Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: fmt.Sprintf("my-service-remote-%v", clusterCount-1),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length to first endpoint is 4
	// Path length from NSE client to other remote endpoint is 8
	require.Equal(t, 8*(clusterCount-1)+4, len(conn.Path.PathSegments))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
}
