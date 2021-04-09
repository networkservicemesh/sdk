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
	"net/url"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// Node is a NSMgr with Forwarder, NSE registry clients
type Node struct {
	t      *testing.T
	domain *Domain

	NSMgr                   *NSMgrEntry
	ForwarderRegistryClient registryapi.NetworkServiceEndpointRegistryClient
	EndpointRegistryClient  registryapi.NetworkServiceEndpointRegistryClient
	NSRegistryClient        registryapi.NetworkServiceRegistryClient
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

	dialOptions := DefaultDialOptions(generatorFunc)

	options := []nsmgr.Option{
		nsmgr.WithName(name),
		nsmgr.WithAuthorizeServer(authorize.NewServer(authorize.Any())),
		nsmgr.WithConnectOptions(
			connect.WithDialTimeout(DialTimeout),
			connect.WithDialOptions(dialOptions...)),
	}

	if n.domain.Registry != nil {
		options = append(options, nsmgr.WithRegistry(n.domain.Registry.URL, dialOptions...))
	}

	if serveURL.Scheme != "unix" {
		options = append(options, nsmgr.WithURL(serveURL.String()))
	}

	entry := &NSMgrEntry{
		Nsmgr: supplyNSMgr(ctx, generatorFunc, options...),
		Name:  name,
		URL:   serveURL,
	}

	serve(ctx, n.t, serveURL, entry.Register)

	log.FromContext(ctx).Infof("Started listening NSMgr %s on %s", name, serveURL.String())

	n.NSMgr = entry
	n.ForwarderRegistryClient = registryclient.NewNetworkServiceEndpointRegistryInterposeClient(ctx, serveURL,
		registryclient.WithDialOptions(dialOptions...))
	n.EndpointRegistryClient = registryclient.NewNetworkServiceEndpointRegistryClient(ctx, serveURL,
		registryclient.WithDialOptions(dialOptions...))
	n.NSRegistryClient = registryclient.NewNetworkServiceRegistryClient(ctx, serveURL,
		registryclient.WithDialOptions(dialOptions...))

	return entry
}

// NewForwarder starts a new forwarder and registers it on the node NSMgr
func (n *Node) NewForwarder(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	generatorFunc token.GeneratorFunc,
	additionalFunctionality ...networkservice.NetworkServiceServer,
) *EndpointEntry {
	if nse.Url == "" {
		nse.Url = n.domain.supplyURL("forwarder").String()
	}

	entry := new(EndpointEntry)
	additionalFunctionality = append(additionalFunctionality,
		clienturl.NewServer(n.NSMgr.URL),
		heal.NewServer(ctx, addressof.NetworkServiceClient(adapters.NewServerToClient(entry))),
		connect.NewServer(ctx,
			client.NewCrossConnectClientFactory(
				client.WithName(nse.Name),
			),
			connect.WithDialTimeout(DialTimeout),
			connect.WithDialOptions(DefaultDialOptions(generatorFunc)...),
		),
	)

	*entry = *n.newEndpoint(ctx, nse, generatorFunc, n.ForwarderRegistryClient, additionalFunctionality...)

	return entry
}

// NewEndpoint starts a new endpoint and registers it on the node NSMgr
func (n *Node) NewEndpoint(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	generatorFunc token.GeneratorFunc,
	additionalFunctionality ...networkservice.NetworkServiceServer,
) *EndpointEntry {
	if nse.Url == "" {
		nse.Url = n.domain.supplyURL("nse").String()
	}

	return n.newEndpoint(ctx, nse, generatorFunc, n.EndpointRegistryClient, additionalFunctionality...)
}

func (n *Node) newEndpoint(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	generatorFunc token.GeneratorFunc,
	registryClient registryapi.NetworkServiceEndpointRegistryClient,
	additionalFunctionality ...networkservice.NetworkServiceServer,
) *EndpointEntry {
	name := nse.Name
	entry := endpoint.NewServer(ctx, generatorFunc,
		endpoint.WithName(name),
		endpoint.WithAdditionalFunctionality(additionalFunctionality...),
	)

	serveURL, err := url.Parse(nse.Url)
	require.NoError(n.t, err)

	serve(ctx, n.t, serveURL, entry.Register)

	reg, err := registryClient.Register(ctx, nse)
	require.NoError(n.t, err)

	nse.Name = reg.Name
	nse.ExpirationTime = reg.ExpirationTime
	nse.NetworkServiceLabels = reg.NetworkServiceLabels

	log.FromContext(ctx).Infof("Started listening endpoint %s on %s", nse.Name, serveURL.String())

	return &EndpointEntry{
		Endpoint: entry,
		Name:     name,
		URL:      serveURL,
	}
}

// NewClient starts a new client and connects it to the node NSMgr
func (n *Node) NewClient(
	ctx context.Context,
	generatorFunc token.GeneratorFunc,
	additionalFunctionality ...networkservice.NetworkServiceClient,
) networkservice.NetworkServiceClient {
	return client.NewClient(
		ctx,
		dial(ctx, n.t, n.NSMgr.URL, generatorFunc),
		client.WithAuthorizeClient(authorize.NewClient(authorize.Any())),
		client.WithAdditionalFunctionality(additionalFunctionality...),
	)
}
