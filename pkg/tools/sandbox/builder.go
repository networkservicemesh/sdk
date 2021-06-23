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

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgrproxy"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/chains/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/chains/proxydns"
	registryconnect "github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// Builder implements builder pattern for building NSM Domain
type Builder struct {
	t   *testing.T
	ctx context.Context

	nodesCount int

	supplyNSMgr         SupplyNSMgrFunc
	supplyNSMgrProxy    SupplyNSMgrProxyFunc
	supplyRegistry      SupplyRegistryFunc
	supplyRegistryProxy SupplyRegistryProxyFunc
	setupNode           SetupNodeFunc

	name                   string
	dnsResolver            dnsresolve.Resolver
	generateTokenFunc      token.GeneratorFunc
	registryExpiryDuration time.Duration

	useUnixSockets bool

	domain *Domain
}

// NewBuilder creates new SandboxBuilder
func NewBuilder(ctx context.Context, t *testing.T) *Builder {
	b := &Builder{
		t:                      t,
		ctx:                    ctx,
		nodesCount:             1,
		supplyNSMgr:            nsmgr.NewServer,
		supplyNSMgrProxy:       nsmgrproxy.NewServer,
		supplyRegistry:         memory.NewServer,
		supplyRegistryProxy:    proxydns.NewServer,
		name:                   "cluster.local",
		dnsResolver:            new(FakeDNSResolver),
		generateTokenFunc:      GenerateTestToken,
		registryExpiryDuration: time.Minute,
	}

	b.setupNode = func(ctx context.Context, node *Node, _ int) {
		SetupDefaultNode(ctx, node, b.supplyNSMgr)
	}

	return b
}

// SetNodesCount sets nodes count
func (b *Builder) SetNodesCount(nodesCount int) *Builder {
	b.nodesCount = nodesCount
	return b
}

// SetNSMgrSupplier replaces default nsmgr supplier to custom function
func (b *Builder) SetNSMgrSupplier(f SupplyNSMgrFunc) *Builder {
	b.supplyNSMgr = f
	return b
}

// SetNSMgrProxySupplier replaces default nsmgr-proxy supplier to custom function
func (b *Builder) SetNSMgrProxySupplier(f SupplyNSMgrProxyFunc) *Builder {
	b.supplyNSMgrProxy = f
	return b
}

// SetRegistrySupplier replaces default memory registry supplier to custom function
func (b *Builder) SetRegistrySupplier(f SupplyRegistryFunc) *Builder {
	b.supplyRegistry = f
	return b
}

// SetRegistryProxySupplier replaces default memory registry supplier to custom function
func (b *Builder) SetRegistryProxySupplier(f SupplyRegistryProxyFunc) *Builder {
	b.supplyRegistryProxy = f
	return b
}

// SetNodeSetup replaces default node setup to custom function
func (b *Builder) SetNodeSetup(f SetupNodeFunc) *Builder {
	require.NotNil(b.t, f)

	b.setupNode = f
	return b
}

// SetDNSDomainName sets DNS domain name for the building NSM domain
func (b *Builder) SetDNSDomainName(name string) *Builder {
	b.name = name
	return b
}

// SetDNSResolver sets DNS resolver for proxy registries
func (b *Builder) SetDNSResolver(d dnsresolve.Resolver) *Builder {
	b.dnsResolver = d
	return b
}

// SetTokenGenerateFunc sets function for the token generation
func (b *Builder) SetTokenGenerateFunc(f token.GeneratorFunc) *Builder {
	b.generateTokenFunc = f
	return b
}

// SetRegistryExpiryDuration replaces registry expiry duration to custom
func (b *Builder) SetRegistryExpiryDuration(registryExpiryDuration time.Duration) *Builder {
	b.registryExpiryDuration = registryExpiryDuration
	return b
}

// UseUnixSockets sets 1 node and mark it to use unix socket to listen on.
func (b *Builder) UseUnixSockets() *Builder {
	require.NotEqual(b.t, "windows", runtime.GOOS, "Unix sockets are not available for windows")

	b.nodesCount = 1
	b.supplyNSMgrProxy = nil
	b.supplyRegistry = nil
	b.supplyRegistryProxy = nil
	b.useUnixSockets = true
	return b
}

// Build builds Domain and Supplier
func (b *Builder) Build() *Domain {
	b.domain = &Domain{
		Name:        b.name,
		DNSResolver: b.dnsResolver,
	}

	if b.useUnixSockets {
		msg := "Unix sockets are available only for local tests with no external registry"
		require.Equal(b.t, b.nodesCount, 1, msg)
		require.Nil(b.t, b.supplyNSMgrProxy, msg)
		require.Nil(b.t, b.supplyRegistry, msg)
		require.Nil(b.t, b.supplyRegistryProxy, msg)

		sockPath, err := ioutil.TempDir(os.TempDir(), "nsm-domain-temp")
		require.NoError(b.t, err)

		go func() {
			<-b.ctx.Done()
			_ = os.RemoveAll(sockPath)
		}()

		b.domain.supplyURL = b.supplyUnixAddress(sockPath, new(int))
	} else {
		b.domain.supplyURL = b.supplyTCPAddress()
	}

	if b.supplyRegistryProxy != nil {
		require.NotNil(b.t, b.supplyNSMgrProxy, "NSMgr proxy supplier should be set if registry proxy supplier is set")
		b.domain.NSMgrProxy = &NSMgrEntry{
			URL: b.domain.supplyURL("nsmgr-proxy"),
		}
	}

	b.domain.RegistryProxy = b.newRegistryProxy()
	b.domain.Registry = b.newRegistry()
	b.domain.NSMgrProxy = b.newNSMgrProxy()
	for i := 0; i < b.nodesCount; i++ {
		b.domain.Nodes = append(b.domain.Nodes, b.newNode(i))
	}

	b.buildDNSServer()

	return b.domain
}

func (b *Builder) supplyUnixAddress(sockPath string, usedAddress *int) func(prefix string) *url.URL {
	return func(prefix string) *url.URL {
		defer func() { *usedAddress++ }()
		return &url.URL{
			Scheme: "unix",
			Path:   fmt.Sprintf("%s/%s_%d.sock", sockPath, prefix, *usedAddress),
		}
	}
}

func (b *Builder) supplyTCPAddress() func(prefix string) *url.URL {
	return func(_ string) *url.URL {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(b.t, err)
		defer func() { _ = l.Close() }()

		return grpcutils.AddressToURL(l.Addr())
	}
}

func (b *Builder) newRegistryProxy() *RegistryEntry {
	if b.supplyRegistryProxy == nil {
		return nil
	}

	registryProxy := b.supplyRegistryProxy(b.ctx, b.dnsResolver, DefaultDialOptions(b.generateTokenFunc)...)
	serveURL := b.domain.supplyURL("reg-proxy")

	serve(b.ctx, b.t, serveURL, registryProxy.Register)

	log.FromContext(b.ctx).Debugf("%s: registry-proxy-dns on: %v", b.name, serveURL)

	return &RegistryEntry{
		URL:      serveURL,
		Registry: registryProxy,
	}
}

func (b *Builder) newRegistry() *RegistryEntry {
	if b.supplyRegistry == nil {
		return nil
	}

	var nsmgrProxyURL *url.URL
	if b.domain.NSMgrProxy != nil {
		nsmgrProxyURL = b.domain.NSMgrProxy.URL
	}

	registry := b.supplyRegistry(b.ctx, b.registryExpiryDuration, nsmgrProxyURL, DefaultDialOptions(b.generateTokenFunc)...)
	serveURL := b.domain.supplyURL("reg")

	serve(b.ctx, b.t, serveURL, registry.Register)

	log.FromContext(b.ctx).Debugf("%s: registry on: %v", b.name, serveURL)

	return &RegistryEntry{
		URL:      serveURL,
		Registry: registry,
	}
}

func (b *Builder) newNSMgrProxy() *NSMgrEntry {
	if b.supplyRegistryProxy == nil {
		return nil
	}

	name := UniqueName("nsmgr-proxy")
	mgr := b.supplyNSMgrProxy(b.ctx,
		b.domain.Registry.URL,
		b.domain.RegistryProxy.URL,
		b.generateTokenFunc,
		nsmgrproxy.WithListenOn(b.domain.NSMgrProxy.URL),
		nsmgrproxy.WithName(name),
		nsmgrproxy.WithConnectOptions(
			connect.WithDialTimeout(DialTimeout),
			connect.WithDialOptions(DefaultDialOptions(b.generateTokenFunc)...)),
		nsmgrproxy.WithRegistryConnectOptions(
			registryconnect.WithDialOptions(DefaultDialOptions(b.generateTokenFunc)...),
		),
	)

	serve(b.ctx, b.t, b.domain.NSMgrProxy.URL, mgr.Register)

	log.FromContext(b.ctx).Debugf("%s: NSMgr proxy %s on: %v", b.name, name, b.domain.NSMgrProxy.URL)

	return &NSMgrEntry{
		Name:  name,
		URL:   b.domain.NSMgrProxy.URL,
		Nsmgr: mgr,
	}
}

func (b *Builder) newNode(nodeNum int) *Node {
	node := &Node{
		t:      b.t,
		domain: b.domain,
	}

	b.setupNode(b.ctx, node, nodeNum)

	require.NotNil(b.t, node.NSMgr, "NSMgr should be set for the node")

	return node
}

func (b *Builder) buildDNSServer() {
	resolver, ok := b.dnsResolver.(*FakeDNSResolver)
	if !ok {
		return
	}

	if b.domain.Registry != nil {
		resolver.AddSRVEntry(b.name, dnsresolve.DefaultRegistryService, b.domain.Registry.URL)
	}
	if b.domain.NSMgrProxy != nil {
		resolver.AddSRVEntry(b.name, dnsresolve.DefaultNsmgrProxyService, b.domain.NSMgrProxy.URL)
	}
}
