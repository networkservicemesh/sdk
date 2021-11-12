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

package nsmgr_test

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/tools/clientinfo"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

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

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService())
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

	const expectedCorefile = ". {\n\tlog\n\treload\n}\nmy.domain1 {\n\tfanout . 8.8.4.4 8.8.8.8\n\tcache\n\tlog\n}\n"

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, dnscontext.NewClient(
		dnscontext.WithChainContext(ctx),
		dnscontext.WithCorefilePath(corefilePath),
		dnscontext.WithResolveConfigPath(resolveConfigPath),
	))

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

func Test_ShouldCorrectlyAddEndpointsWithSameNames(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetRegistryExpiryDuration(sandbox.RegistryExpiryDuration).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	// 1. Add endpoints
	var nseRegs [2]*registry.NetworkServiceEndpoint
	var nses [2]*sandbox.EndpointEntry
	for i := range nseRegs {
		nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService())
		require.NoError(t, err)

		nseRegs[i] = defaultRegistryEndpoint(nsReg.Name)
		nseRegs[i].NetworkServiceNames[0] = nsReg.Name

		nses[i] = domain.Nodes[0].NewEndpoint(ctx, nseRegs[i], sandbox.GenerateTestToken)
	}

	// 2. Wait for refresh
	<-time.After(sandbox.RegistryExpiryDuration)

	// 3. Request
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	for _, nseReg := range nseRegs {
		_, err := nsc.Request(ctx, defaultRequest(nseReg.NetworkServiceNames[0]))
		require.NoError(t, err)
	}

	// 3. Delete endpoints
	for i, nseReg := range nseRegs {
		_, err := nses[i].Unregister(ctx, nseReg)
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

	nsReg := defaultRegistryService()
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
		SetRegistryExpiryDuration(sandbox.RegistryExpiryDuration).
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

	conn, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 4, len(conn.Path.PathSegments))
	require.Equal(t, "p2p forwarder", conn.GetPath().GetPathSegments()[2].Name)
}
