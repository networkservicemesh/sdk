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
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
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

	NSMgr *NSMgrEntry
}

// URL returns node NSMgr URL
func (n *Node) URL() *url.URL {
	u := new(url.URL)
	*u = *n.NSMgr.URL
	return u
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
		Name: name,
		URL:  serveURL,
	}
	entry.restartableServer = newRestartableServer(ctx, n.t, entry.URL, func(ctx context.Context) {
		entry.Nsmgr = supplyNSMgr(ctx, generatorFunc, options...)
		serve(ctx, n.t, entry.URL, entry.Register)

		log.FromContext(ctx).Debugf("%s: NSMgr %s on %v", n.domain.Name, name, serveURL)
	})

	n.NSMgr = entry

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

	dialOptions := DefaultDialOptions(generatorFunc)

	entry := new(EndpointEntry)
	additionalFunctionality = append(additionalFunctionality,
		clienturl.NewServer(n.URL()),
		heal.NewServer(ctx,
			heal.WithOnHeal(addressof.NetworkServiceClient(adapters.NewServerToClient(entry))),
			heal.WithOnRestore(heal.OnRestoreIgnore)),
		connect.NewServer(ctx,
			client.NewClientFactory(
				client.WithName(nse.Name),
				client.WithAdditionalFunctionality(
					mechanismtranslation.NewClient(),
				),
			),
			connect.WithDialTimeout(DialTimeout),
			connect.WithDialOptions(dialOptions...),
		),
	)

	*entry = *n.newEndpoint(
		ctx,
		nse,
		generatorFunc,
		registryclient.NewNetworkServiceEndpointRegistryInterposeClient(ctx, n.URL(),
			registryclient.WithDialOptions(dialOptions...)),
		additionalFunctionality...,
	)

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

	return n.newEndpoint(
		ctx,
		nse,
		generatorFunc,
		registryclient.NewNetworkServiceEndpointRegistryClient(ctx, n.URL(),
			registryclient.WithDialOptions(DefaultDialOptions(generatorFunc)...)),
		additionalFunctionality...,
	)
}

func (n *Node) newEndpoint(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	generatorFunc token.GeneratorFunc,
	registryClient registryapi.NetworkServiceEndpointRegistryClient,
	additionalFunctionality ...networkservice.NetworkServiceServer,
) *EndpointEntry {
	name := nse.Name
	ep := endpoint.NewServer(ctx, generatorFunc,
		endpoint.WithName(name),
		endpoint.WithAdditionalFunctionality(additionalFunctionality...),
	)

	serveURL, err := url.Parse(nse.Url)
	require.NoError(n.t, err)

	serve(ctx, n.t, serveURL, ep.Register)

	reg, err := registryClient.Register(ctx, nse)
	require.NoError(n.t, err)

	nse.Name = reg.Name
	nse.ExpirationTime = reg.ExpirationTime
	nse.NetworkServiceLabels = reg.NetworkServiceLabels

	log.FromContext(ctx).Debugf("%s: endpoint %s on %v", n.domain.Name, nse.Name, serveURL)

	return &EndpointEntry{
		Name:                                 name,
		URL:                                  serveURL,
		Endpoint:                             ep,
		NetworkServiceEndpointRegistryClient: registryClient,
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
		n.URL(),
		client.WithDialOptions(DefaultDialOptions(generatorFunc)...),
		client.WithDialTimeout(DialTimeout),
		client.WithAuthorizeClient(authorize.NewClient(authorize.Any())),
		client.WithAdditionalFunctionality(additionalFunctionality...),
	)
}
