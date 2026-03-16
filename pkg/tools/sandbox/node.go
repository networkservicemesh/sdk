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

package sandbox

import (
	"context"
	"net/url"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/retry"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/roundrobin"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/registry/common/recvfd"
	"github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	authmonitor "github.com/networkservicemesh/sdk/pkg/tools/monitorconnection/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// Node is a NSMgr with Forwarder, NSE registry clients
type Node struct {
	t      *testing.T
	domain *Domain

	NSMgr      *NSMgrEntry
	Forwarders map[string]*EndpointEntry
}

// NewNSMgr creates a new NSMgr
func (n *Node) NewNSMgr(
	ctx context.Context,
	name string,
	serveURL *url.URL,
	generatorFunc token.GeneratorFunc,
	supplyNSMgr SupplyNSMgrFunc,
) *NSMgrEntry {
	if serveURL == nil {
		serveURL = n.domain.supplyURL("nsmgr")
	}

	dialOptions := DialOptions(WithTokenGenerator(generatorFunc))

	options := []nsmgr.Option{
		nsmgr.WithName(name),
		nsmgr.WithAuthorizeServer(authorize.NewServer(authorize.Any())),
		nsmgr.WithAuthorizeMonitorConnectionServer(authmonitor.NewMonitorConnectionServer(authmonitor.Any())),
		nsmgr.WithDialOptions(dialOptions...),
		nsmgr.WithDialTimeout(DialTimeout),
	}

	if n.domain.Registry != nil {
		options = append(options, nsmgr.WithRegistry(CloneURL(n.domain.Registry.URL)))
	}

	if serveURL.Scheme != "unix" {
		options = append(options, nsmgr.WithURL(serveURL.String()))
	}

	entry := &NSMgrEntry{
		Name: name,
		URL:  serveURL,
	}
	entry.restartableServer = newRestartableServer(ctx, n.t, entry.URL, func(ctx context.Context) {
		entry.Nsmgr = supplyNSMgr(ctx, generatorFunc, options...)
		serve(ctx, n.t, entry.URL, entry.Register)

		log.FromContext(ctx).Infof("%s: NSMgr %s serve on %v", n.domain.Name, name, serveURL)
	}, nil)

	n.NSMgr = entry

	return entry
}

// NewForwarder starts a new forwarder and registers it on the node NSMgr
func (n *Node) NewForwarder(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	generatorFunc token.GeneratorFunc,
	opts ...ForwarderOption,
) *EndpointEntry {
	var serveURL *url.URL
	var err error
	if nse.Url == "" {
		serveURL = n.domain.supplyURL("forwarder")
		nse.Url = serveURL.String()
	} else {
		serveURL, err = url.Parse(nse.Url)
		require.NoError(n.t, err)
	}

	var serverOptions = &forwarderOptions{}
	for _, opt := range opts {
		opt(serverOptions)
	}

	nseClone := nse.Clone()
	dialOptions := DialOptions(WithTokenGenerator(generatorFunc))

	entry := &EndpointEntry{
		Name: nse.Name,
		URL:  serveURL,
	}
	nseClient := chain.NewNetworkServiceEndpointRegistryClient(
		registryclient.NewNetworkServiceEndpointRegistryClient(ctx,
			registryclient.WithClientURL(CloneURL(n.NSMgr.URL)),
			registryclient.WithNSEAdditionalFunctionality(recvfd.NewNetworkServiceEndpointRegistryClient()),
			registryclient.WithDialOptions(dialOptions...),
		),
	)
	nsClient := registryclient.NewNetworkServiceRegistryClient(ctx,
		registryclient.WithClientURL(CloneURL(n.NSMgr.URL)),
		registryclient.WithDialOptions(dialOptions...))
	entry.restartableServer = newRestartableServer(ctx, n.t, entry.URL, func(ctx context.Context) {
		entry.Endpoint = endpoint.NewServer(ctx, generatorFunc,
			endpoint.WithName(entry.Name),
			endpoint.WithAdditionalFunctionality(
				append(
					append([]networkservice.NetworkServiceServer{
						discover.NewServer(nsClient, nseClient),
						roundrobin.NewServer(),
					}, serverOptions.additionalFunctionalityServer...),
					connect.NewServer(
						client.NewClient(
							ctx,
							client.WithName(entry.Name),
							client.WithAdditionalFunctionality(
								append([]networkservice.NetworkServiceClient{
									mechanismtranslation.NewClient(),
								}, serverOptions.additionalFunctionalityClient...)...,
							),
							client.WithDialOptions(dialOptions...),
							client.WithDialTimeout(DialTimeout),
							client.WithoutRefresh(),
						),
					),
				)...,
			),
		)
		serve(ctx, n.t, entry.URL, entry.Endpoint.Register)

		log.FromContext(ctx).Infof("%s: forwarder %s serve on %v", n.domain.Name, nse.Name, serveURL)

		entry.NetworkServiceEndpointRegistryClient = registryclient.NewNetworkServiceEndpointRegistryClient(
			ctx,
			registryclient.WithClientURL(CloneURL(n.NSMgr.URL)),
			registryclient.WithDialOptions(dialOptions...),
		)

		n.registerEndpoint(ctx, nse, nseClone, entry.NetworkServiceEndpointRegistryClient)
		n.Forwarders[entry.Name] = entry
	}, func() {
		delete(n.Forwarders, entry.Name)
	})

	return entry
}

// NewEndpoint starts a new endpoint and registers it on the node NSMgr
func (n *Node) NewEndpoint(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	generatorFunc token.GeneratorFunc,
	additionalFunctionality ...networkservice.NetworkServiceServer,
) *EndpointEntry {
	var serveURL *url.URL
	var err error
	if nse.Url == "" {
		serveURL = n.domain.supplyURL("nse")
		nse.Url = serveURL.String()
	} else {
		serveURL, err = url.Parse(nse.Url)
		require.NoError(n.t, err)
	}

	nseClone := nse.Clone()
	dialOptions := DialOptions(WithTokenGenerator(generatorFunc))

	entry := &EndpointEntry{
		Name: nse.Name,
		URL:  serveURL,
	}
	entry.restartableServer = newRestartableServer(ctx, n.t, entry.URL, func(ctx context.Context) {
		entry.Endpoint = endpoint.NewServer(ctx, generatorFunc,
			endpoint.WithName(entry.Name),
			endpoint.WithAdditionalFunctionality(additionalFunctionality...),
		)

		serve(ctx, n.t, entry.URL, entry.Endpoint.Register)

		log.FromContext(ctx).Infof("%s: NSE %s serve on %v", n.domain.Name, nse.Name, serveURL)

		entry.NetworkServiceEndpointRegistryClient = registryclient.NewNetworkServiceEndpointRegistryClient(
			ctx,
			registryclient.WithClientURL(CloneURL(n.NSMgr.URL)),
			registryclient.WithDialOptions(dialOptions...),
			registryclient.WithNSEAdditionalFunctionality(sendfd.NewNetworkServiceEndpointRegistryClient()),
		)

		n.registerEndpoint(ctx, nse, nseClone, entry.NetworkServiceEndpointRegistryClient)
	}, nil)

	return entry
}

func (n *Node) registerEndpoint(
	ctx context.Context,
	nse, nseClone *registryapi.NetworkServiceEndpoint,
	registryClient registryapi.NetworkServiceEndpointRegistryClient,
) {
	reg, err := registryClient.Register(ctx, nseClone.Clone())
	require.NoError(n.t, err)

	nse.Name = reg.Name
	nse.ExpirationTime = reg.ExpirationTime
	nse.NetworkServiceLabels = reg.NetworkServiceLabels
}

// NewClient starts a new client and connects it to the node NSMgr
func (n *Node) NewClient(
	ctx context.Context,
	generatorFunc token.GeneratorFunc,
	additionalOpts ...client.Option,
) networkservice.NetworkServiceClient {
	opts := []client.Option{
		client.WithClientURL(CloneURL(n.NSMgr.URL)),
		client.WithDialOptions(DialOptions(WithTokenGenerator(generatorFunc))...),
		client.WithAuthorizeClient(authorize.NewClient(authorize.Any())),
		client.WithHealClient(heal.NewClient(ctx)),
		client.WithDialTimeout(DialTimeout),
	}

	opts = append(opts, additionalOpts...)
	return retry.NewClient(client.NewClient(
		ctx,
		opts...,
	))
}
