// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

//go:build !windows
// +build !windows

package nsmgr_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/edwarnicke/genericsync"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/cache"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/dnsconfigs"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/fanout"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/noloop"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/norecursion"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/searches"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func requireIPv4Lookup(ctx context.Context, t *testing.T, r *net.Resolver, host, expected string) {
	addrs, err := r.LookupIP(ctx, "ip4", host)
	require.NoError(t, err)
	require.Len(t, addrs, 1)
	require.Equal(t, expected, addrs[0].String())
}

func Test_DNSUsecase(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.GetName())

	nse := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	dnsConfigsMap := new(genericsync.Map[string, []*networkservice.DNSConfig])
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithAdditionalFunctionality(dnscontext.NewClient(
		dnscontext.WithChainContext(ctx),
		dnscontext.WithDNSConfigsMap(dnsConfigsMap),
	)))

	dnsConfigs := []*networkservice.DNSConfig{
		{
			DnsServerIps:  []string{"127.0.0.1:40053"},
			SearchDomains: []string{"com"},
		},
	}

	// DNS server on nse side
	dnsRecords := new(genericsync.Map[string, []net.IP])
	dnsRecords.Store("my.domain.", []net.IP{net.ParseIP("4.4.4.4")})
	dnsRecords.Store("my.domain.com.", []net.IP{net.ParseIP("5.5.5.5")})
	dnsutils.ListenAndServe(ctx, memory.NewDNSHandler(dnsRecords), ":40053")

	// DNS server on nsc side
	clientDNSHandler := next.NewDNSHandler(
		dnsconfigs.NewDNSHandler(dnsConfigsMap),
		searches.NewDNSHandler(),
		noloop.NewDNSHandler(),
		norecursion.NewDNSHandler(),
		cache.NewDNSHandler(),
		fanout.NewDNSHandler(),
	)
	dnsutils.ListenAndServe(ctx, clientDNSHandler, ":50053")

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: nsReg.GetName(),
			Context: &networkservice.ConnectionContext{
				DnsContext: &networkservice.DNSContext{
					Configs: dnsConfigs,
				},
			},
			Labels: make(map[string]string),
		},
	}

	resolver := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, "127.0.0.1:50053")
		},
	}

	_, err = resolver.LookupIP(ctx, "ip4", "my.domain")
	require.Error(t, err)

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)

	healthCheck := func() bool {
		_, e := resolver.LookupIP(ctx, "ip4", "my.domain")
		return e == nil
	}
	// To ensure that DNS records sent with nsc.Request are ready
	require.Eventually(t, healthCheck, time.Second, 10*time.Millisecond)

	requireIPv4Lookup(ctx, t, &resolver, "my.domain", "4.4.4.4")
	requireIPv4Lookup(ctx, t, &resolver, "my.domain.com", "5.5.5.5")

	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	_, err = nse.Unregister(ctx, nseReg)
	require.NoError(t, err)
}
