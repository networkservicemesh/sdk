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
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgrproxy"
	"github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/registry/chains/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/chains/proxydns"
	registryconnect "github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
	"github.com/networkservicemesh/sdk/pkg/tools/tracing"
)

// Builder implements builder pattern for building NSM Domain
type Builder struct {
	require                *require.Assertions
	resources              []context.CancelFunc
	nodesCount             int
	nodesConfig            []*NodeConfig
	DNSDomainName          string
	resolver               dnsresolve.Resolver
	supplyNSMgr            SupplyNSMgrFunc
	supplyNSMgrProxy       SupplyNSMgrProxyFunc
	supplyRegistry         SupplyRegistryFunc
	supplyRegistryProxy    SupplyRegistryProxyFunc
	setupNode              SetupNodeFunc
	generateTokenFunc      token.GeneratorFunc
	registryExpiryDuration time.Duration
	ctx                    context.Context
	t                      *testing.T

	useUnixSockets bool
	sockPath       string
	usedAddress    int
}

// NewBuilder creates new SandboxBuilder
func NewBuilder(t *testing.T) *Builder {
	return &Builder{
		nodesCount:             1,
		require:                require.New(t),
		supplyNSMgr:            nsmgr.NewServer,
		DNSDomainName:          "cluster.local",
		supplyRegistry:         memory.NewServer,
		supplyRegistryProxy:    proxydns.NewServer,
		supplyNSMgrProxy:       nsmgrproxy.NewServer,
		setupNode:              defaultSetupNode(t),
		generateTokenFunc:      GenerateTestToken,
		registryExpiryDuration: time.Minute,
		resolver:               new(FakeDNSResolver),
		t:                      t,

		useUnixSockets: false,
	}
}

// Build builds Domain and Supplier
func (b *Builder) Build() *Domain {
	ctx := b.ctx
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		b.resources = append(b.resources, cancel)
	}
	ctx = log.Join(ctx, log.Empty())

	domain := new(Domain)

	if b.useUnixSockets {
		var err error
		b.sockPath, err = ioutil.TempDir(os.TempDir(), "nsm-domain-temp")
		b.require.NoError(err)
		domain.domainTemp = b.sockPath
	}

	domain.RegistryProxy = b.newRegistryProxy(ctx)

	var nsmgrProxyURL = new(url.URL)

	if domain.RegistryProxy == nil {
		domain.Registry = b.newRegistry(ctx, nil)
	} else {
		domain.Registry = b.newRegistry(ctx, nsmgrProxyURL)
	}

	if domain.RegistryProxy != nil {
		domain.NSMgrProxy = b.newNSMgrProxy(ctx, domain.Registry.URL, domain.RegistryProxy.URL)
		*nsmgrProxyURL = *domain.NSMgrProxy.URL
	}

	var registryURL *url.URL
	if domain.Registry != nil {
		registryURL = domain.Registry.URL
	}

	for i := 0; i < b.nodesCount; i++ {
		domain.Nodes = append(domain.Nodes, b.newNode(ctx, registryURL, b.nodesConfig[i]))
	}

	domain.resources, b.resources = b.resources, nil

	b.t.Cleanup(domain.cleanup)

	b.buildDNSServer(domain)
	domain.Name = b.DNSDomainName

	return domain
}

func (b *Builder) buildDNSServer(d *Domain) {
	resolver, ok := b.resolver.(*FakeDNSResolver)
	if !ok {
		return
	}

	if d.Registry != nil {
		resolver.AddSRVEntry(b.DNSDomainName, dnsresolve.DefaultRegistryService, d.Registry.URL)
	}
	if d.NSMgrProxy != nil {
		resolver.AddSRVEntry(b.DNSDomainName, dnsresolve.DefaultNsmgrProxyService, d.NSMgrProxy.URL)
	}
}

// SetContext sets default context for all chains
func (b *Builder) SetContext(ctx context.Context) *Builder {
	b.ctx = ctx
	b.SetCustomConfig([]*NodeConfig{})
	return b
}

// SetCustomConfig sets custom configuration for nodes
func (b *Builder) SetCustomConfig(config []*NodeConfig) *Builder {
	oldConfig := b.nodesConfig
	b.nodesConfig = nil

	for i := 0; i < b.nodesCount; i++ {
		nodeConfig := &NodeConfig{}
		if i < len(config) && config[i] != nil {
			*nodeConfig = *oldConfig[i]
		}

		customConfig := &NodeConfig{}
		if i < len(config) && config[i] != nil {
			*customConfig = *config[i]
		}

		if customConfig.NsmgrCtx != nil {
			nodeConfig.NsmgrCtx = customConfig.NsmgrCtx
		} else if nodeConfig.NsmgrCtx == nil {
			nodeConfig.NsmgrCtx = b.ctx
		}

		if customConfig.NsmgrGenerateTokenFunc != nil {
			nodeConfig.NsmgrGenerateTokenFunc = customConfig.NsmgrGenerateTokenFunc
		} else if nodeConfig.NsmgrGenerateTokenFunc == nil {
			nodeConfig.NsmgrGenerateTokenFunc = b.generateTokenFunc
		}

		if customConfig.ForwarderCtx != nil {
			nodeConfig.ForwarderCtx = customConfig.ForwarderCtx
		} else if nodeConfig.ForwarderCtx == nil {
			nodeConfig.ForwarderCtx = b.ctx
		}

		if customConfig.ForwarderGenerateTokenFunc != nil {
			nodeConfig.ForwarderGenerateTokenFunc = customConfig.ForwarderGenerateTokenFunc
		} else if nodeConfig.ForwarderGenerateTokenFunc == nil {
			nodeConfig.ForwarderGenerateTokenFunc = b.generateTokenFunc
		}

		b.nodesConfig = append(b.nodesConfig, nodeConfig)
	}
	return b
}

// SetNodesCount sets nodes count
func (b *Builder) SetNodesCount(nodesCount int) *Builder {
	b.nodesCount = nodesCount
	b.SetCustomConfig([]*NodeConfig{})
	return b
}

// UseUnixSockets sets 1 node and mark it to use unix socket to listen on.
func (b *Builder) UseUnixSockets() *Builder {
	if runtime.GOOS == "windows" {
		panic("Unix sockets are not available for windows")
	}
	b.useUnixSockets = true
	return b
}

// SetDNSResolver sets DNS resolver for proxy registries
func (b *Builder) SetDNSResolver(d dnsresolve.Resolver) *Builder {
	b.resolver = d
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

// SetRegistryExpiryDuration replaces registry expiry duration to custom
func (b *Builder) SetRegistryExpiryDuration(registryExpiryDuration time.Duration) *Builder {
	b.registryExpiryDuration = registryExpiryDuration
	return b
}

func (b *Builder) dialContext(ctx context.Context, u *url.URL) *grpc.ClientConn {
	conn, err := grpc.DialContext(ctx, grpcutils.URLToTarget(u), DefaultDialOptions(b.generateTokenFunc)...)
	b.resources = append(b.resources, func() {
		_ = conn.Close()
	})
	b.require.NoError(err, "Can not dial to %v", u)
	return conn
}

func (b *Builder) newNSMgrProxy(ctx context.Context, registryURL, registryProxyURL *url.URL) *EndpointEntry {
	if b.supplyRegistryProxy == nil {
		return nil
	}
	name := "nsmgr-proxy-" + uuid.New().String()
	serveURL := grpcutils.TargetToURL(b.newAddress("nsmgr-proxy"))
	mgr := b.supplyNSMgrProxy(ctx,
		registryURL,
		registryProxyURL,
		b.generateTokenFunc,
		nsmgrproxy.WithListenOn(serveURL),
		nsmgrproxy.WithName(name),
		nsmgrproxy.WithConnectOptions(
			connect.WithDialTimeout(DialTimeout),
			connect.WithDialOptions(DefaultDialOptions(b.generateTokenFunc)...)),
		nsmgrproxy.WithRegistryConnectOptions(
			registryconnect.WithDialOptions(DefaultDialOptions(b.generateTokenFunc)...),
		),
	)

	serve(ctx, serveURL, mgr.Register)
	log.FromContext(ctx).Debugf("%v: %v listen on: %v", b.DNSDomainName, name, serveURL)
	return &EndpointEntry{
		Endpoint: mgr,
		URL:      serveURL,
	}
}

// NewNSMgr - starts new Network Service Manager
func (b *Builder) NewNSMgr(ctx context.Context, node *Node, address string, registryURL *url.URL, generateTokenFunc token.GeneratorFunc) (entry *NSMgrEntry, resources []context.CancelFunc) {
	nsmgrCtx, nsmgrCancel := context.WithCancel(ctx)
	b.resources = append(b.resources, nsmgrCancel)

	entry = b.newNSMgr(nsmgrCtx, address, registryURL, generateTokenFunc)

	b.SetupRegistryClients(ctx, node)

	resources, b.resources = b.resources, nil
	return
}

func (b *Builder) newNSMgr(ctx context.Context, address string, registryURL *url.URL, generateTokenFunc token.GeneratorFunc) *NSMgrEntry {
	if b.supplyNSMgr == nil {
		return nil
	}

	var serveURL *url.URL
	if b.useUnixSockets {
		serveURL = grpcutils.TargetToURL(address)
	} else {
		listener, err := net.Listen("tcp", address)
		b.require.NoError(err)
		serveURL = grpcutils.AddressToURL(listener.Addr())
		b.require.NoError(listener.Close())
	}

	nsmgrName := "nsmgr-" + uuid.New().String()

	options := []nsmgr.Option{
		nsmgr.WithName(nsmgrName),
		nsmgr.WithAuthorizeServer(authorize.NewServer(authorize.Any())),
		nsmgr.WithConnectOptions(
			connect.WithDialTimeout(DialTimeout),
			connect.WithDialOptions(DefaultDialOptions(generateTokenFunc)...)),
	}

	if registryURL != nil {
		options = append(options, nsmgr.WithRegistry(registryURL, DefaultDialOptions(generateTokenFunc)...))
	}

	if serveURL.Scheme == "tcp" {
		options = append(options, nsmgr.WithURL(serveURL.String()))
	}

	mgr := b.supplyNSMgr(ctx, generateTokenFunc, options...)

	serve(ctx, serveURL, mgr.Register)
	log.FromContext(ctx).Debugf("%v: %v listen on: %v", b.DNSDomainName, nsmgrName, serveURL)
	return &NSMgrEntry{
		URL:   serveURL,
		Nsmgr: mgr,
	}
}

func serve(ctx context.Context, u *url.URL, register func(server *grpc.Server)) {
	server := grpc.NewServer(tracing.WithTracing()...)
	register(server)
	errCh := grpcutils.ListenAndServe(ctx, u, server)
	go func() {
		select {
		case <-ctx.Done():
			log.FromContext(ctx).Debugf("Stop serve: %v", u.String())
			return
		case err := <-errCh:
			if err != nil {
				log.FromContext(ctx).Fatalf("An error during serve: %v", err.Error())
			}
		}
	}()
}

func (b *Builder) newRegistryProxy(ctx context.Context) *RegistryEntry {
	if b.supplyRegistryProxy == nil {
		return nil
	}
	result := b.supplyRegistryProxy(ctx, b.resolver, DefaultDialOptions(b.generateTokenFunc)...)
	serveURL := grpcutils.TargetToURL(b.newAddress("reg-proxy"))
	serve(ctx, serveURL, result.Register)
	log.FromContext(ctx).Debugf("%v: registry-proxy-dns listen on: %v", b.DNSDomainName, serveURL)
	return &RegistryEntry{
		URL:      serveURL,
		Registry: result,
	}
}

func (b *Builder) newRegistry(ctx context.Context, proxyRegistryURL *url.URL) *RegistryEntry {
	if b.supplyRegistry == nil {
		return nil
	}
	result := b.supplyRegistry(ctx, b.registryExpiryDuration, proxyRegistryURL, DefaultDialOptions(b.generateTokenFunc)...)
	serveURL := grpcutils.TargetToURL(b.newAddress("reg"))
	serve(ctx, serveURL, result.Register)
	log.FromContext(ctx).Debugf("%v: registry listen on: %v", b.DNSDomainName, serveURL)
	return &RegistryEntry{
		URL:      serveURL,
		Registry: result,
	}
}

func (b *Builder) newNode(ctx context.Context, registryURL *url.URL, nodeConfig *NodeConfig) *Node {
	address := b.newAddress("nsmgr")
	nsmgrEntry := b.newNSMgr(nodeConfig.NsmgrCtx, address, registryURL, nodeConfig.NsmgrGenerateTokenFunc)

	node := &Node{
		ctx:   b.ctx,
		NSMgr: nsmgrEntry,
	}

	b.SetupRegistryClients(ctx, node)

	if b.setupNode != nil {
		b.setupNode(ctx, node, nodeConfig)
	}

	return node
}

// SetupRegistryClients - creates Network Service Registry Clients
func (b *Builder) SetupRegistryClients(ctx context.Context, node *Node) {
	if node.NSMgr == nil {
		return
	}

	dialOptions := DefaultDialOptions(b.generateTokenFunc)

	node.ForwarderRegistryClient = client.NewNetworkServiceEndpointRegistryInterposeClient(ctx, node.NSMgr.URL, client.WithDialOptions(dialOptions...))
	node.EndpointRegistryClient = client.NewNetworkServiceEndpointRegistryClient(ctx, node.NSMgr.URL, client.WithDialOptions(dialOptions...))
	node.NSRegistryClient = client.NewNetworkServiceRegistryClient(ctx, node.NSMgr.URL, client.WithDialOptions(dialOptions...))
}

// newAddress - will return a new public address, if unixSockets are used prefix will be used to make uniq files.
func (b *Builder) newAddress(prefix string) string {
	if !b.useUnixSockets {
		return "127.0.0.1:0"
	}

	b.usedAddress++
	return fmt.Sprintf("unix:%s/%s_%d.sock", b.sockPath, prefix, b.usedAddress)
}

func defaultSetupNode(t *testing.T) SetupNodeFunc {
	return func(ctx context.Context, node *Node, nodeConfig *NodeConfig) {
		nseReg := &registryapi.NetworkServiceEndpoint{
			Name: "forwarder",
		}
		_, err := node.NewForwarder(nodeConfig.ForwarderCtx, nseReg, nodeConfig.ForwarderGenerateTokenFunc)
		require.NoError(t, err)
	}
}
