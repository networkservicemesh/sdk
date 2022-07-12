// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

package nsmgr_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/excludedprefixes"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/point2pointipam"
	"github.com/networkservicemesh/sdk/pkg/tools/clientinfo"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func Test_DNSUsecase(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*7)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	nse := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, dnscontext.NewServer(
		&networkservice.DNSConfig{
			DnsServerIps:  []string{"8.8.8.8"},
			SearchDomains: []string{"my.domain1"},
		},
		&networkservice.DNSConfig{
			DnsServerIps:  []string{"8.8.4.4"},
			SearchDomains: []string{"my.domain1"},
		},
	))

	corefilePath := filepath.Join(t.TempDir(), "corefile")
	resolveConfigPath := filepath.Join(t.TempDir(), "resolv.conf")

	err = ioutil.WriteFile(resolveConfigPath, []byte("nameserver 8.8.4.4\nsearch example.com\n"), os.ModePerm)
	require.NoError(t, err)

	const expectedCorefile = ". {\n\tfanout . 8.8.4.4\n\tlog\n\treload\n\tcache {\n\t\tdenial 0\n\t}\n}\nmy.domain1 {\n\tfanout . 8.8.4.4 8.8.8.8\n\tlog\n\tcache {\n\t\tdenial 0\n\t}\n}"

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithAdditionalFunctionality(dnscontext.NewClient(
		dnscontext.WithChainContext(ctx),
		dnscontext.WithCorefilePath(corefilePath),
		dnscontext.WithResolveConfigPath(resolveConfigPath),
	)))

	conn, err := nsc.Request(ctx, defaultRequest(nsReg.Name))
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		// #nosec
		b, readFileErr := ioutil.ReadFile(corefilePath)
		if readFileErr != nil {
			return false
		}
		return string(b) == expectedCorefile
	}, time.Second, time.Millisecond*100)

	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	_, err = nse.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func Test_AwareNSEs(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	_, ipNet, err := net.ParseCIDR("172.16.0.96/29")
	require.NoError(t, err)

	const count = 3
	var nseRegs [count]*registry.NetworkServiceEndpoint
	var nses [count]*sandbox.EndpointEntry
	var requests [count]*networkservice.NetworkServiceRequest

	ns1 := defaultRegistryService("my-ns-1")
	ns2 := defaultRegistryService("my-ns-2")

	nsurl1, err := url.Parse(fmt.Sprintf("kernel://%s?%s=%s", ns1.Name, "color", "red"))
	require.NoError(t, err)

	nsurl2, err := url.Parse(fmt.Sprintf("kernel://%s?%s=%s", ns2.Name, "color", "red"))
	require.NoError(t, err)

	nsInfo := [count]struct {
		ns         *registry.NetworkService
		labelKey   string
		labelValue string
	}{
		{
			ns:         ns1,
			labelKey:   "color",
			labelValue: "red",
		},
		{
			ns:         ns2,
			labelKey:   "color",
			labelValue: "red",
		},
		{
			ns:         ns1,
			labelKey:   "day",
			labelValue: "friday",
		},
	}

	for i := 0; i < count; i++ {
		nseRegs[i] = &registry.NetworkServiceEndpoint{
			Name:                fmt.Sprintf("nse-%s", uuid.New().String()),
			NetworkServiceNames: []string{nsInfo[i].ns.Name},
			NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
				nsInfo[i].ns.Name: {
					Labels: map[string]string{
						nsInfo[i].labelKey: nsInfo[i].labelValue,
					},
				},
			},
		}

		nses[i] = domain.Nodes[0].NewEndpoint(ctx, nseRegs[i], sandbox.GenerateTestToken, point2pointipam.NewServer(ipNet))

		requests[i] = &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id:             fmt.Sprint(i),
				NetworkService: nsInfo[i].ns.Name,
				Context:        &networkservice.ConnectionContext{},
				Mechanism:      &networkservice.Mechanism{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
				Labels: map[string]string{
					nsInfo[i].labelKey: nsInfo[i].labelValue,
				},
			},
		}

		nsInfo[i].ns.Matches = append(nsInfo[i].ns.Matches,
			&registry.Match{
				SourceSelector: map[string]string{nsInfo[i].labelKey: nsInfo[i].labelValue},
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{nsInfo[i].labelKey: nsInfo[i].labelValue},
					},
				},
			},
		)
	}

	_, err = nsRegistryClient.Register(ctx, ns1)
	require.NoError(t, err)
	_, err = nsRegistryClient.Register(ctx, ns2)
	require.NoError(t, err)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithAdditionalFunctionality(
		excludedprefixes.NewClient(excludedprefixes.WithAwarenessGroups(
			[][]*url.URL{
				{nsurl1, nsurl2},
			},
		))))

	var conns [count]*networkservice.Connection
	for i := 0; i < count; i++ {
		conns[i], err = nsc.Request(ctx, requests[i])
		require.NoError(t, err)
		require.Equal(t, conns[0].NetworkServiceEndpointName, nses[0].Name)
	}

	srcIP1 := conns[0].GetContext().GetIpContext().GetSrcIpAddrs()
	srcIP2 := conns[1].GetContext().GetIpContext().GetSrcIpAddrs()
	srcIP3 := conns[2].GetContext().GetIpContext().GetSrcIpAddrs()

	require.Equal(t, srcIP1[0], srcIP2[0])
	require.NotEqual(t, srcIP1[0], srcIP3[0])
	require.NotEqual(t, srcIP2[0], srcIP3[0])

	for i := 0; i < count; i++ {
		_, err = nsc.Close(ctx, conns[i])
		require.NoError(t, err)
	}

	for i := 0; i < count; i++ {
		_, err = nses[i].Unregister(ctx, nseRegs[i])
		require.NoError(t, err)
	}
}

func Test_ShouldParseNetworkServiceLabelsTemplate(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	const (
		testEnvName             = "NODE_NAME"
		testEnvValue            = "testValue"
		destinationTestKey      = `nodeName`
		destinationTestTemplate = `{{.nodeName}}`
	)

	err := os.Setenv(testEnvName, testEnvValue)
	require.NoError(t, err)

	want := map[string]string{}
	clientinfo.AddClientInfo(ctx, want)

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := defaultRegistryService(t.Name())
	nsReg.Matches = []*registry.Match{
		{
			Routes: []*registry.Destination{
				{
					DestinationSelector: map[string]string{
						destinationTestKey: destinationTestTemplate,
					},
				},
			},
		},
	}

	nsReg, err = nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)
	nseReg.NetworkServiceLabels = map[string]*registry.NetworkServiceLabels{nsReg.Name: {}}

	nse := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)
	require.NoError(t, err)

	req := defaultRequest(nsReg.Name)

	conn, err := nsc.Request(ctx, req)
	require.NoError(t, err)

	// Test for connection labels setting
	require.Equal(t, want, conn.Labels)
	// Test for endpoint labels setting
	require.Equal(t, want, nseReg.NetworkServiceLabels[nsReg.Name].Labels)

	_, err = nse.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func Test_UsecasePoint2MultiPoint(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, _ int) {
			node.NewNSMgr(ctx, "nsmgr", nil, sandbox.GenerateTestToken, nsmgr.NewServer)
		}).
		SetRegistryExpiryDuration(time.Second).
		Build()

	domain.Nodes[0].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
		Name:                "p2mp forwarder",
		NetworkServiceNames: []string{"forwarder"},
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			"forwarder": {
				Labels: map[string]string{
					"p2mp": "true",
				},
			},
		},
	}, sandbox.GenerateTestToken)

	domain.Nodes[0].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
		Name:                "p2p forwarder",
		NetworkServiceNames: []string{"forwarder"},
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			"forwarder": {
				Labels: map[string]string{
					"p2p": "true",
				},
			},
		},
	}, sandbox.GenerateTestToken)

	domain.Nodes[0].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
		Name:                "special forwarder",
		NetworkServiceNames: []string{"forwarder"},
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			"forwarder": {
				Labels: map[string]string{
					"special": "true",
				},
			},
		},
	}, sandbox.GenerateTestToken)

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	_, err := nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-ns",
		Matches: []*registry.Match{
			{
				SourceSelector: map[string]string{},
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{},
					},
				},
				Metadata: &registry.Metadata{
					Labels: map[string]string{
						"p2mp": "true",
					},
				},
			},
		},
	})
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "my-nse-1",
		NetworkServiceNames: []string{"my-ns"},
	}

	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest("my-ns")

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 4, len(conn.Path.PathSegments))
	require.Equal(t, "p2mp forwarder", conn.GetPath().GetPathSegments()[2].Name)

	_, err = nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-ns",
		Matches: []*registry.Match{
			{
				SourceSelector: map[string]string{},
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{},
					},
				},
				Metadata: &registry.Metadata{
					Labels: map[string]string{
						// no labels
					},
				},
			},
		},
	})
	require.NoError(t, err)

	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	conn, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 4, len(conn.Path.PathSegments))
	require.Equal(t, "p2p forwarder", conn.GetPath().GetPathSegments()[2].Name)
}
func Test_RemoteUsecase_Point2MultiPoint(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	const nodeCount = 2

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(nodeCount).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, _ int) {
			node.NewNSMgr(ctx, "nsmgr", nil, sandbox.GenerateTestToken, nsmgr.NewServer)
		}).
		SetRegistryExpiryDuration(time.Second).
		Build()

	for i := 0; i < nodeCount; i++ {
		domain.Nodes[i].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
			Name:                "p2mp forwarder-" + fmt.Sprint(i),
			NetworkServiceNames: []string{"forwarder"},
			NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
				"forwarder": {
					Labels: map[string]string{
						"p2mp": "true",
					},
				},
			},
		}, sandbox.GenerateTestToken)

		domain.Nodes[i].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
			Name:                "p2p forwarder-" + fmt.Sprint(i),
			NetworkServiceNames: []string{"forwarder"},
			NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
				"forwarder": {
					Labels: map[string]string{
						"p2p": "true",
					},
				},
			},
		}, sandbox.GenerateTestToken)

		domain.Nodes[i].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
			Name:                "special forwarder-" + fmt.Sprint(i),
			NetworkServiceNames: []string{"forwarder"},
			NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
				"forwarder": {
					Labels: map[string]string{
						"special": "true",
					},
				},
			},
		}, sandbox.GenerateTestToken)
	}
	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	_, err := nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-ns",
		Matches: []*registry.Match{
			{
				Metadata: &registry.Metadata{
					Labels: map[string]string{
						"p2mp": "true",
					},
				},
			},
		},
	})
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "my-nse-1",
		NetworkServiceNames: []string{"my-ns"},
	}

	domain.Nodes[1].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest("my-ns")

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 6, len(conn.Path.PathSegments))
	require.Equal(t, "p2mp forwarder-0", conn.GetPath().GetPathSegments()[2].Name)
	require.Equal(t, "p2mp forwarder-1", conn.GetPath().GetPathSegments()[4].Name)

	_, err = nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-ns",
		Matches: []*registry.Match{
			{
				Metadata: &registry.Metadata{
					Labels: map[string]string{
						// no labels
					},
				},
			},
		},
	})
	require.NoError(t, err)

	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	conn, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 6, len(conn.Path.PathSegments))
	require.Equal(t, "p2p forwarder-0", conn.GetPath().GetPathSegments()[2].Name)
	require.Equal(t, "p2p forwarder-1", conn.GetPath().GetPathSegments()[4].Name)
}
