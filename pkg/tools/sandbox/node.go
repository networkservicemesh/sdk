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

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// Node is a NSMgr with Forwarder, NSE registry clients
type Node struct {
	ctx                     context.Context
	NSMgr                   *NSMgrEntry
	Forwarder               []*EndpointEntry
	ForwarderRegistryClient registryapi.NetworkServiceEndpointRegistryClient
	EndpointRegistryClient  registryapi.NetworkServiceEndpointRegistryClient
	NSRegistryClient        registryapi.NetworkServiceRegistryClient
}

// NewForwarder starts a new forwarder and registers it on the node NSMgr
func (n *Node) NewForwarder(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	generatorFunc token.GeneratorFunc,
	additionalFunctionality ...networkservice.NetworkServiceServer,
) (*EndpointEntry, error) {
	ep := new(EndpointEntry)
	additionalFunctionality = append(additionalFunctionality,
		clienturl.NewServer(n.NSMgr.URL),
		heal.NewServer(ctx, addressof.NetworkServiceClient(adapters.NewServerToClient(ep))),
		connect.NewServer(ctx,
			client.NewClientFactory(
				client.WithName(nse.Name),
				client.WithAdditionalFunctionality(
					mechanismtranslation.NewClient(),
				),
			),
			connect.WithDialTimeout(DialTimeout),
			connect.WithDialOptions(DefaultDialOptions(generatorFunc)...),
		),
	)

	entry, err := n.newEndpoint(ctx, nse, generatorFunc, n.ForwarderRegistryClient, additionalFunctionality...)
	if err != nil {
		return nil, err
	}
	*ep = *entry
	n.Forwarder = append(n.Forwarder, ep)
	return ep, nil
}

// NewEndpoint starts a new endpoint and registers it on the node NSMgr
func (n *Node) NewEndpoint(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	generatorFunc token.GeneratorFunc,
	additionalFunctionality ...networkservice.NetworkServiceServer,
) (*EndpointEntry, error) {
	return n.newEndpoint(ctx, nse, generatorFunc, n.EndpointRegistryClient, additionalFunctionality...)
}

func (n *Node) newEndpoint(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	generatorFunc token.GeneratorFunc,
	registryClient registryapi.NetworkServiceEndpointRegistryClient,
	additionalFunctionality ...networkservice.NetworkServiceServer,
) (_ *EndpointEntry, err error) {
	// 1. Create endpoint server
	ep := endpoint.NewServer(ctx, generatorFunc,
		endpoint.WithName(nse.Name),
		endpoint.WithAdditionalFunctionality(additionalFunctionality...),
	)

	// 2. Start listening on URL
	u := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	if nse.Url != "" {
		u, err = url.Parse(nse.Url)
		if err != nil {
			return nil, err
		}
	}

	serve(ctx, u, ep.Register)

	nse.Url = u.String()

	// 3. Register with the node registry client
	var reg *registryapi.NetworkServiceEndpoint
	if reg, err = registryClient.Register(ctx, nse); err != nil {
		return nil, err
	}

	nse.Name = reg.Name
	nse.ExpirationTime = reg.ExpirationTime
	nse.NetworkServiceLabels = reg.NetworkServiceLabels

	log.FromContext(ctx).Debugf("Started listen endpoint %s on %s.", nse.Name, u.String())

	return &EndpointEntry{Endpoint: ep, URL: u}, nil
}

// NewClient starts a new client and connects it to the node NSMgr
func (n *Node) NewClient(
	ctx context.Context,
	generatorFunc token.GeneratorFunc,
	additionalFunctionality ...networkservice.NetworkServiceClient,
) networkservice.NetworkServiceClient {
	return client.NewClient(
		ctx,
		n.NSMgr.URL,
		client.WithDialOptions(DefaultDialOptions(generatorFunc)...),
		client.WithAuthorizeClient(authorize.NewClient(authorize.Any())),
		client.WithAdditionalFunctionality(additionalFunctionality...),
	)
}
