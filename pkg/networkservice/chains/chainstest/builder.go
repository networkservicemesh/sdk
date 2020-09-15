// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package chainstest

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/registry"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	interpose_reg "github.com/networkservicemesh/sdk/pkg/registry/common/interpose"
	adapter_registry "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// DomainBuilder implements builder pattern for building Domain and Supplier
type DomainBuilder struct {
	require           *require.Assertions
	resources         []context.CancelFunc
	nodesCount        int
	supplyForwarder   SupplyForwarderFunc
	supplyNSMgr       SupplyNSMgrFunc
	supplyRegistry    SupplyRegistryFunc
	supplyNSE         SupplyEndpointFunc
	supplyNSC         SupplyClientFunc
	generateTokenFunc token.GeneratorFunc
	timeout           time.Duration
}

// NewDomainBuilder creates new DomainBuilder
func NewDomainBuilder(t *testing.T) *DomainBuilder {
	return &DomainBuilder{
		nodesCount:        1,
		require:           require.New(t),
		supplyNSMgr:       nsmgr.NewServer,
		supplyForwarder:   supplyDummyForwarder,
		supplyRegistry:    supplyMemoryRegistry,
		supplyNSE:         supplyDummyNSE,
		supplyNSC:         client.NewClient,
		generateTokenFunc: tokenGenerator,
		timeout:           time.Second * 15,
	}
}

// Build builds Domain and Supplier
func (b *DomainBuilder) Build() (*Domain, *Supplier) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	b.resources = append(b.resources, cancel)
	domain := &Domain{}
	reg, u := b.newRegistry(ctx, b.supplyRegistry)
	domain.Registry = &RegistryEntry{
		Registry: reg,
		URL:      u,
	}
	for i := 0; i < b.nodesCount; i++ {
		var node = new(Node)
		mgr, u := b.newNSMgr(ctx, domain.Registry.URL)
		node.NSMgr = &NSMgrEntry{
			Nsmgr: mgr,
			URL:   u,
		}
		forwarderName := "cross-nse" + uuid.New().String()
		forwarder, u := b.newCrossConnectNSE(ctx, forwarderName, node.NSMgr.URL)
		node.Forwarder = &EndpointEntry{
			Endpoint: forwarder,
			URL:      u,
		}
		forwarderRegistrationClient := chain.NewNetworkServiceEndpointRegistryClient(
			interpose_reg.NewNetworkServiceEndpointRegistryClient(),
			adapter_registry.NetworkServiceEndpointServerToClient(node.NSMgr.NetworkServiceEndpointRegistryServer()),
		)
		_, err := forwarderRegistrationClient.Register(context.Background(), &registryapi.NetworkServiceEndpoint{
			Url:  node.Forwarder.URL.String(),
			Name: forwarderName,
		})
		b.require.NoError(err)
		domain.Nodes = append(domain.Nodes, node)
	}
	supplier := &Supplier{
		resources:    b.resources,
		supplyNSC:    b.supplyNSC,
		supplyNSE:    b.supplyNSE,
		tokenGenFunc: b.generateTokenFunc,
		require:      b.require,
		dialContext:  b.dialContext,
		ctx:          ctx,
	}
	b.resources = nil
	return domain, supplier
}

// SetTimeout sets timeout for test
func (b *DomainBuilder) SetTimeout(timeout time.Duration) *DomainBuilder {
	b.timeout = timeout
	return b
}

// SetNodesCount sets nodes count
func (b *DomainBuilder) SetNodesCount(nodesCount int) *DomainBuilder {
	b.nodesCount = nodesCount
	return b
}

// SetTokenGenerateFunc sets function for the token generation
func (b *DomainBuilder) SetTokenGenerateFunc(f token.GeneratorFunc) *DomainBuilder {
	b.generateTokenFunc = f
	return b
}

// SetRegistrySupplier sets registry supplier
func (b *DomainBuilder) SetRegistrySupplier(f SupplyRegistryFunc) *DomainBuilder {
	b.supplyRegistry = f
	return b
}

// SetForwarderSupplier sets forwarder supplier
func (b *DomainBuilder) SetForwarderSupplier(f SupplyForwarderFunc) *DomainBuilder {
	b.supplyForwarder = f
	return b
}

// SetNSMgrSupplier sets SupplyNSMgrFunc
func (b *DomainBuilder) SetNSMgrSupplier(f SupplyNSMgrFunc) *DomainBuilder {
	b.supplyNSMgr = f
	return b
}

func (b *DomainBuilder) dialContext(ctx context.Context, u *url.URL) *grpc.ClientConn {
	conn, err := grpc.DialContext(ctx, grpcutils.URLToTarget(u),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	b.resources = append(b.resources, func() {
		_ = conn.Close()
	})
	b.require.NoError(err, "Can not dial to", u)
	return conn
}

func (b *DomainBuilder) newNSMgr(ctx context.Context, registryURL *url.URL) (nsmgr.Nsmgr, *url.URL) {
	var registryCC *grpc.ClientConn
	if registryURL != nil {
		registryCC = b.dialContext(ctx, registryURL)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	b.require.NoError(err)
	serveURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:" + fmt.Sprint(listener.Addr().(*net.TCPAddr).Port)}
	b.require.NoError(listener.Close())

	nsmgrReg := &registryapi.NetworkServiceEndpoint{
		Name: "nsmgr-" + uuid.New().String(),
		Url:  serveURL.String(),
	}

	mgr := b.supplyNSMgr(ctx, nsmgrReg, authorize.NewServer(), b.generateTokenFunc, registryCC, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))

	serve(ctx, serveURL, mgr.Register)
	log.Entry(ctx).Infof("NSMgr %v listen on: %v", nsmgrReg.Name, serveURL)
	return mgr, serveURL
}

func serve(ctx context.Context, u *url.URL, register func(server *grpc.Server)) {
	server := grpc.NewServer()
	register(server)
	errCh := grpcutils.ListenAndServe(ctx, u, server)
	go func() {
		select {
		case <-ctx.Done():
			log.Entry(ctx).Infof("Stop serve: %v", u.String())
			return
		case err := <-errCh:
			log.Entry(ctx).Fatalf("An error during serve: %v", err.Error())
		}
	}()
}
func (b *DomainBuilder) newCrossConnectNSE(ctx context.Context, name string, connectTo *url.URL) (endpoint.Endpoint, *url.URL) {
	crossNSE := b.supplyForwarder(ctx, name, b.generateTokenFunc, connectTo, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	serveURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	serve(ctx, serveURL, crossNSE.Register)
	log.Entry(ctx).Infof("%v listen on: %v", name, serveURL)
	return crossNSE, serveURL
}

func (b *DomainBuilder) newRegistry(ctx context.Context, supply SupplyRegistryFunc) (registry.Registry, *url.URL) {
	result := supply()
	serveURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	serve(ctx, serveURL, result.Register)
	log.Entry(ctx).Infof("Registry listen on: %v", serveURL)
	return result, serveURL
}

func tokenGenerator(_ credentials.AuthInfo) (tokenValue string, expireTime time.Time, err error) {
	return "TestToken", time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC), nil
}

func supplyDummyNSE(ctx context.Context, registration *registryapi.NetworkServiceEndpoint, genToken token.GeneratorFunc) endpoint.Endpoint {
	return endpoint.NewServer(ctx,
		registration.Name,
		authorize.NewServer(),
		genToken)
}
func supplyMemoryRegistry() registry.Registry {
	return registry.NewServer(chain.NewNetworkServiceRegistryServer(memory.NewNetworkServiceRegistryServer()), chain.NewNetworkServiceEndpointRegistryServer(memory.NewNetworkServiceEndpointRegistryServer()))
}

func supplyDummyForwarder(ctx context.Context, name string, generateToken token.GeneratorFunc, connectTo *url.URL, dialOptions ...grpc.DialOption) endpoint.Endpoint {
	var result endpoint.Endpoint
	result = endpoint.NewServer(ctx,
		name,
		authorize.NewServer(),
		generateToken,
		// Statically set the url we use to the unix file socket for the NSMgr
		clienturl.NewServer(connectTo),
		connect.NewServer(
			ctx,
			client.NewClientFactory(
				name,
				// What to call onHeal
				addressof.NetworkServiceClient(adapters.NewServerToClient(result)),
				tokenGenerator,
			),
			grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		))

	return result
}
