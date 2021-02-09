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

package sandbox

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgrproxy"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/registry/chains/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/chains/proxydns"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/opentracing"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

const (
	defaultContextTimeout         = time.Second * 15
	defaultRegistryExpiryDuration = 500 * time.Millisecond
)

// Builder implements builder pattern for building NSM Domain
type Builder struct {
	require             *require.Assertions
	resources           []context.CancelFunc
	nodesCount          int
	DNSDomainName       string
	Resolver            dnsresolve.Resolver
	supplyNSMgr         SupplyNSMgrFunc
	supplyNSMgrProxy    SupplyNSMgrProxyFunc
	supplyRegistry      SupplyRegistryFunc
	supplyRegistryProxy SupplyRegistryProxyFunc
	setupNode           SetupNodeFunc
	generateTokenFunc   token.GeneratorFunc
	ctx                 context.Context
}

// NewBuilder creates new SandboxBuilder
func NewBuilder(t *testing.T) *Builder {
	return &Builder{
		nodesCount:          1,
		require:             require.New(t),
		Resolver:            net.DefaultResolver,
		supplyNSMgr:         nsmgr.NewServer,
		DNSDomainName:       "cluster.local",
		supplyRegistry:      memory.NewServer,
		supplyRegistryProxy: proxydns.NewServer,
		supplyNSMgrProxy:    nsmgrproxy.NewServer,
		setupNode:           defaultSetupNode(t),
		generateTokenFunc:   GenerateTestToken,
	}
}

// Build builds Domain and Supplier
func (b *Builder) Build() *Domain {
	ctx := b.ctx
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), defaultContextTimeout)
		b.resources = append(b.resources, cancel)
	}
	ctx = log.Join(ctx, log.Empty())

	domain := new(Domain)
	domain.NSMgrProxy = b.newNSMgrProxy(ctx)
	if domain.NSMgrProxy == nil {
		domain.RegistryProxy = b.newRegistryProxy(ctx, &url.URL{})
	} else {
		domain.RegistryProxy = b.newRegistryProxy(ctx, domain.NSMgrProxy.URL)
	}
	if domain.RegistryProxy == nil {
		domain.Registry = b.newRegistry(ctx, nil)
	} else {
		domain.Registry = b.newRegistry(ctx, domain.RegistryProxy.URL)
	}

	for i := 0; i < b.nodesCount; i++ {
		domain.Nodes = append(domain.Nodes, b.newNode(ctx, domain.Registry.URL))
	}

	domain.resources, b.resources = b.resources, nil

	return domain
}

// SetContext sets context for all chains
func (b *Builder) SetContext(ctx context.Context) *Builder {
	b.ctx = ctx
	return b
}

// SetNodesCount sets nodes count
func (b *Builder) SetNodesCount(nodesCount int) *Builder {
	b.nodesCount = nodesCount
	return b
}

// SetDNSResolver sets DNS resolver for proxy registries
func (b *Builder) SetDNSResolver(d dnsresolve.Resolver) *Builder {
	b.Resolver = d
	return b
}

// SetTokenGenerateFunc sets function for the token generation
func (b *Builder) SetTokenGenerateFunc(f token.GeneratorFunc) *Builder {
	b.generateTokenFunc = f
	return b
}

// SetRegistryProxySupplier replaces default memory registry supplier to custom function
func (b *Builder) SetRegistryProxySupplier(f SupplyRegistryProxyFunc) *Builder {
	b.supplyRegistryProxy = f
	return b
}

// SetRegistrySupplier replaces default memory registry supplier to custom function
func (b *Builder) SetRegistrySupplier(f SupplyRegistryFunc) *Builder {
	b.supplyRegistry = f
	return b
}

// SetDNSDomainName sets DNS domain name for the building NSM domain
func (b *Builder) SetDNSDomainName(name string) *Builder {
	b.DNSDomainName = name
	return b
}

// SetNSMgrProxySupplier replaces default nsmgr-proxy supplier to custom function
func (b *Builder) SetNSMgrProxySupplier(f SupplyNSMgrProxyFunc) *Builder {
	b.supplyNSMgrProxy = f
	return b
}

// SetNSMgrSupplier replaces default nsmgr supplier to custom function
func (b *Builder) SetNSMgrSupplier(f SupplyNSMgrFunc) *Builder {
	b.supplyNSMgr = f
	return b
}

// SetNodeSetup replaces default node setup to custom function
func (b *Builder) SetNodeSetup(f SetupNodeFunc) *Builder {
	b.setupNode = f
	return b
}

func (b *Builder) dialContext(ctx context.Context, u *url.URL) *grpc.ClientConn {
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

func (b *Builder) newNSMgrProxy(ctx context.Context) *EndpointEntry {
	if b.supplyRegistryProxy == nil {
		return nil
	}
	name := "nsmgr-proxy-" + uuid.New().String()
	mgr := b.supplyNSMgrProxy(ctx, name, b.generateTokenFunc, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	serveURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	serve(ctx, serveURL, mgr.Register)
	log.FromContext(ctx).Infof("%v listen on: %v", name, serveURL)
	return &EndpointEntry{
		Endpoint: mgr,
		URL:      serveURL,
	}
}

func (b *Builder) newNSMgr(ctx context.Context, registryURL *url.URL) *NSMgrEntry {
	if b.supplyNSMgr == nil {
		panic("nodes without managers are not supported")
	}
	var registryCC *grpc.ClientConn
	if registryURL != nil {
		registryCC = b.dialContext(ctx, registryURL)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	b.require.NoError(err)
	serveURL := grpcutils.AddressToURL(listener.Addr())
	b.require.NoError(listener.Close())

	nsmgrReg := &registryapi.NetworkServiceEndpoint{
		Name: "nsmgr-" + uuid.New().String(),
		Url:  serveURL.String(),
	}

	mgr := b.supplyNSMgr(ctx, nsmgrReg, authorize.NewServer(), b.generateTokenFunc, registryCC, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))

	serve(ctx, serveURL, mgr.Register)
	log.FromContext(ctx).Infof("%v listen on: %v", nsmgrReg.Name, serveURL)
	return &NSMgrEntry{
		URL:   serveURL,
		Nsmgr: mgr,
	}
}

func serve(ctx context.Context, u *url.URL, register func(server *grpc.Server)) {
	server := grpc.NewServer(opentracing.WithTracing()...)
	register(server)
	errCh := grpcutils.ListenAndServe(ctx, u, server)
	go func() {
		select {
		case <-ctx.Done():
			log.FromContext(ctx).Infof("Stop serve: %v", u.String())
			return
		case err := <-errCh:
			if err != nil {
				log.FromContext(ctx).Fatalf("An error during serve: %v", err.Error())
			}
		}
	}()
}

func (b *Builder) newRegistryProxy(ctx context.Context, nsmgrProxyURL *url.URL) *RegistryEntry {
	if b.supplyRegistryProxy == nil {
		return nil
	}
	result := b.supplyRegistryProxy(ctx, b.Resolver, b.DNSDomainName, nsmgrProxyURL, grpc.WithInsecure(), grpc.WithBlock())
	serveURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	serve(ctx, serveURL, result.Register)
	log.FromContext(ctx).Infof("registry-proxy-dns listen on: %v", serveURL)
	return &RegistryEntry{
		URL:      serveURL,
		Registry: result,
	}
}

func (b *Builder) newRegistry(ctx context.Context, proxyRegistryURL *url.URL) *RegistryEntry {
	if b.supplyRegistry == nil {
		return nil
	}
	result := b.supplyRegistry(ctx, defaultRegistryExpiryDuration, proxyRegistryURL, grpc.WithInsecure(), grpc.WithBlock())
	serveURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	serve(ctx, serveURL, result.Register)
	log.FromContext(ctx).Infof("Registry listen on: %v", serveURL)
	return &RegistryEntry{
		URL:      serveURL,
		Registry: result,
	}
}

func (b *Builder) newNode(ctx context.Context, registryURL *url.URL) *Node {
	nsmgrEntry := b.newNSMgr(ctx, registryURL)
	nsmgrCC := b.dialContext(ctx, nsmgrEntry.URL)

	node := &Node{
		ctx:                     b.ctx,
		NSMgr:                   nsmgrEntry,
		ForwarderRegistryClient: client.NewNetworkServiceEndpointRegistryInterposeClient(ctx, nsmgrCC),
		EndpointRegistryClient:  client.NewNetworkServiceEndpointRegistryClient(ctx, nsmgrCC),
		NSRegistryClient:        client.NewNetworkServiceRegistryClient(nsmgrCC),
	}

	if b.setupNode != nil {
		b.setupNode(ctx, node)
	}

	return node
}

func defaultSetupNode(t *testing.T) SetupNodeFunc {
	return func(ctx context.Context, node *Node) {
		nseReg := &registryapi.NetworkServiceEndpoint{
			Name: uuid.New().String(),
		}
		_, err := node.NewForwarder(ctx, nseReg, GenerateTestToken)
		require.NoError(t, err)
	}
}
