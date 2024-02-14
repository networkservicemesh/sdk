// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2024 Cisco and/or its affiliates.
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
	"net"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgrproxy"
	"github.com/networkservicemesh/sdk/pkg/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/chains/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/chains/proxydns"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	dnsmemory "github.com/networkservicemesh/sdk/pkg/tools/dnsutils/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
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
	supplyDNSServer     SupplyDNSServerFunc
	setupNode           SetupNodeFunc

	name                      string
	generateTokenFunc         token.GeneratorFunc
	registryDefaultExpiration time.Duration

	dnsServer      *DNSServerEntry
	useUnixSockets bool

	domain *Domain
}

func newRegistryMemoryServer(ctx context.Context, tokenGenerator token.GeneratorFunc, defaultExpiration time.Duration, nsmgrProxyURL, proxyRegistryURL *url.URL, options ...grpc.DialOption) registry.Registry {
	return memory.NewServer(
		ctx,
		tokenGenerator,
		memory.WithDefaultExpiration(defaultExpiration),
		memory.WithNSMgrProxyURL(nsmgrProxyURL),
		memory.WithProxyRegistryURL(proxyRegistryURL),
		memory.WithDialOptions(options...))
}

// NewBuilder creates new SandboxBuilder
func NewBuilder(ctx context.Context, t *testing.T) *Builder {
	b := &Builder{
		t:                         t,
		ctx:                       ctx,
		nodesCount:                1,
		supplyNSMgr:               nsmgr.NewServer,
		supplyRegistry:            newRegistryMemoryServer,
		name:                      "cluster.local",
		generateTokenFunc:         GenerateTestToken,
		registryDefaultExpiration: time.Minute,
	}

	b.setupNode = func(ctx context.Context, node *Node, _ int) {
		SetupDefaultNode(ctx, b.generateTokenFunc, node, b.supplyNSMgr)
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

// SetupDefaultDNSServer sets default dns handler for the domain
func (b *Builder) SetupDefaultDNSServer() *Builder {
	return b.SetDNSServerSupplier(func(ctx context.Context) dnsutils.Handler {
		return nil
	})
}

// EnableInterdomain enables interdomain components for the NSM domain
func (b *Builder) EnableInterdomain() *Builder {
	b.supplyNSMgrProxy = nsmgrproxy.NewServer
	b.supplyRegistryProxy = proxydns.NewServer
	return b.SetupDefaultDNSServer()
}

// SetDNSServerSupplier sets handler for the dns server
func (b *Builder) SetDNSServerSupplier(f SupplyDNSServerFunc) *Builder {
	b.supplyDNSServer = f
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

// SetName sets DNS domain name for the building NSM domain
func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

// SetTokenGenerateFunc sets function for the token generation
func (b *Builder) SetTokenGenerateFunc(f token.GeneratorFunc) *Builder {
	b.generateTokenFunc = f
	return b
}

// SetRegistryDefaultExpiration sets default expiration for endpoints
func (b *Builder) SetRegistryDefaultExpiration(d time.Duration) *Builder {
	b.registryDefaultExpiration = d
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
		Name: b.name,
	}

	if b.useUnixSockets {
		msg := "Unix sockets are available only for local tests with no external registry"
		require.Equal(b.t, b.nodesCount, 1, msg)
		require.Nil(b.t, b.supplyNSMgrProxy, msg)
		require.Nil(b.t, b.supplyRegistry, msg)
		require.Nil(b.t, b.supplyRegistryProxy, msg)

		sockPath, err := os.MkdirTemp(os.TempDir(), "nsm-domain-temp")
		require.NoError(b.t, err)

		go func() {
			<-b.ctx.Done()
			_ = os.RemoveAll(sockPath)
		}()

		b.domain.supplyURL = b.supplyUnixAddress(sockPath, new(int))
	} else {
		b.domain.supplyURL = b.supplyTCPAddress()
	}

	if b.supplyRegistry != nil {
		b.domain.Registry = &RegistryEntry{
			URL: b.domain.supplyURL("reg"),
		}
	}
	if b.supplyNSMgrProxy != nil {
		b.domain.NSMgrProxy = &EndpointEntry{
			URL: b.domain.supplyURL("nsmgr-proxy"),
		}
	}

	b.domain.DNSServer = b.newDNSServer()
	b.domain.RegistryProxy = b.newRegistryProxy()
	b.domain.NSMgrProxy = b.newNSMgrProxy()
	b.domain.Registry = b.newRegistry()
	for i := 0; i < b.nodesCount; i++ {
		b.domain.Nodes = append(b.domain.Nodes, b.newNode(i))
	}

	b.addDNSRecords()

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

func (b *Builder) newDNSServer() *DNSServerEntry {
	if b.supplyDNSServer == nil {
		return nil
	}
	if b.dnsServer != nil {
		return b.dnsServer
	}
	entry := &DNSServerEntry{
		URL: b.domain.supplyURL("dns-server"),
	}

	entry.restartableServer = newRestartableServer(b.ctx, b.t, entry.URL, func(ctx context.Context) {
		var handlers = []dnsutils.Handler{
			dnsmemory.NewDNSHandlerWithOptions(
				dnsmemory.WithIPRecords(&entry.IPRecords),
				dnsmemory.WithSRVRecords(&entry.SRVRecords),
			),
		}
		if additionalFunctionality := b.supplyDNSServer(ctx); additionalFunctionality != nil {
			handlers = append(handlers, additionalFunctionality)
		}

		entry.Handler = next.NewDNSHandler(
			handlers...,
		)

		dnsutils.ListenAndServe(ctx, entry.Handler, entry.URL.Host)

		log.FromContext(ctx).Infof("%s: dns-seever serving on: %v", b.name, entry.URL)
	})

	return entry
}

func (b *Builder) newRegistryProxy() *RegistryEntry {
	if b.supplyRegistryProxy == nil || b.domain.DNSServer == nil {
		return nil
	}

	var entry = &RegistryEntry{
		URL: b.domain.supplyURL("reg-proxy"),
	}

	var nsmgrProxyURL *url.URL

	if b.domain.NSMgrProxy != nil {
		nsmgrProxyURL = b.domain.NSMgrProxy.URL
	}

	var dialer net.Dialer
	var resolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, b.domain.DNSServer.URL.Host)
		},
	}

	entry.restartableServer = newRestartableServer(b.ctx, b.t, entry.URL, func(ctx context.Context) {
		entry.Registry = b.supplyRegistryProxy(
			ctx,
			b.generateTokenFunc,
			resolver,
			proxydns.WithDialOptions(DialOptions(WithTokenGenerator(b.generateTokenFunc))...),
			proxydns.WithRegistryURL(CloneURL(b.domain.Registry.URL)),
			proxydns.WithNSMgrProxyURL(CloneURL(nsmgrProxyURL)),
		)
		serve(ctx, b.t, entry.URL, entry.Register)

		log.FromContext(ctx).Infof("%s: registry-proxy-dns serve on: %v", b.name, entry.URL)
	})

	return entry
}

func (b *Builder) newRegistry() *RegistryEntry {
	if b.supplyRegistry == nil {
		return nil
	}

	var nsmgrProxyURL, registryProxyURL *url.URL
	if b.domain.NSMgrProxy != nil {
		nsmgrProxyURL = CloneURL(b.domain.NSMgrProxy.URL)
	}
	if b.domain.RegistryProxy != nil {
		registryProxyURL = CloneURL(b.domain.RegistryProxy.URL)
	}

	entry := &RegistryEntry{
		URL: b.domain.Registry.URL,
	}
	entry.restartableServer = newRestartableServer(b.ctx, b.t, entry.URL, func(ctx context.Context) {
		entry.Registry = b.supplyRegistry(
			ctx,
			b.generateTokenFunc,
			b.registryDefaultExpiration,
			nsmgrProxyURL,
			registryProxyURL,
			DialOptions(WithTokenGenerator(b.generateTokenFunc))...,
		)
		serve(ctx, b.t, entry.URL, entry.Register)

		log.FromContext(ctx).Infof("%s: registry serve on: %v", b.name, entry.URL)
	})

	return entry
}

func (b *Builder) newNSMgrProxy() *EndpointEntry {
	if b.supplyNSMgrProxy == nil {
		return nil
	}

	entry := &EndpointEntry{
		Name: UniqueName("nsmgr-proxy"),
		URL:  b.domain.NSMgrProxy.URL,
	}

	entry.restartableServer = newRestartableServer(b.ctx, b.t, entry.URL, func(ctx context.Context) {
		dialOptions := DialOptions(WithTokenGenerator(b.generateTokenFunc))
		entry.Endpoint = b.supplyNSMgrProxy(ctx,
			CloneURL(b.domain.RegistryProxy.URL),
			b.generateTokenFunc,
			nsmgrproxy.WithListenOn(entry.URL),
			nsmgrproxy.WithName(entry.Name),
			nsmgrproxy.WithRegistryURL(b.domain.Registry.URL),
			nsmgrproxy.WithDialOptions(dialOptions...),
		)
		serve(ctx, b.t, entry.URL, entry.Endpoint.Register)

		log.FromContext(ctx).Infof("%s: NSMgr proxy %s serve on: %v", b.name, entry.Name, entry.URL)
	})

	return entry
}

func (b *Builder) newNode(nodeNum int) *Node {
	node := &Node{
		t:          b.t,
		domain:     b.domain,
		Forwarders: make(map[string]*EndpointEntry),
	}

	b.setupNode(b.ctx, node, nodeNum)

	require.NotNil(b.t, node.NSMgr, "NSMgr should be set for the node")

	return node
}

func (b *Builder) addDNSRecords() {
	if b.domain.DNSServer == nil {
		return
	}
	var domain = b.domain
	b.addRecord("external-dns", domain.DNSServer.URL)
	if domain.Registry != nil {
		b.addRecord("registry", domain.Registry.URL)
	}
	if domain.NSMgrProxy != nil {
		b.addRecord("nsmgr-proxy", domain.NSMgrProxy.URL)
	}
	if domain.RegistryProxy != nil {
		b.addRecord("registry-proxy", domain.RegistryProxy.URL)
	}
	for i, node := range b.domain.Nodes {
		if node.NSMgr != nil {
			b.addRecord(fmt.Sprintf("nsmgr-%v", i), node.NSMgr.URL)
		}
		for name, fwd := range node.Forwarders {
			b.addRecord(fmt.Sprintf("fwd-%v-%v", name, i), fwd.URL)
		}
	}
}

func (b *Builder) addRecord(name string, u *url.URL) {
	name = fmt.Sprintf("%v.%v.", name, b.domain.Name)
	log.FromContext(b.ctx).Infof("Adding srv, aaaa, records for name %v, dns server: %v", name, b.domain.DNSServer.URL.String())
	var ip = net.ParseIP(u.Hostname())
	var port, _ = strconv.Atoi(u.Port())
	b.domain.DNSServer.IPRecords.Store(name, []net.IP{ip})
	b.domain.DNSServer.SRVRecords.Store(name, []*net.TCPAddr{
		{
			IP:   ip,
			Port: port,
		},
	})
}
