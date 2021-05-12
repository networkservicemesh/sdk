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

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// Node is a NSMgr with API to add and register NSs, Forwarders, NSEs
type Node struct {
	ctx                context.Context
	nsmgrCC            grpc.ClientConnInterface
	nseRegistryClients nseRegistryClientMap

	NSMgr            *NSMgrEntry
	Forwarder        []*EndpointEntry
	NSRegistryClient registryapi.NetworkServiceRegistryClient
}

// NewForwarder starts a new forwarder and registers it on the node NSMgr with its own registry client
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
			client.NewCrossConnectClientFactory(
				client.WithName(nse.Name),
			),
			connect.WithDialTimeout(DialTimeout),
			connect.WithDialOptions(DefaultDialOptions(generatorFunc)...),
		),
	)

	registryClient := registryclient.NewNetworkServiceEndpointRegistryInterposeClient(ctx, n.nsmgrCC)

	entry, err := n.newEndpoint(ctx, nse, generatorFunc, registryClient, additionalFunctionality...)
	if err != nil {
		return nil, err
	}
	*ep = *entry

	n.Forwarder = append(n.Forwarder, ep)

	return ep, nil
}

// NewEndpoint starts a new endpoint and registers it on the node NSMgr with its own registry client
func (n *Node) NewEndpoint(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	generatorFunc token.GeneratorFunc,
	additionalFunctionality ...networkservice.NetworkServiceServer,
) (*EndpointEntry, error) {
	registryClient := registryclient.NewNetworkServiceEndpointRegistryClient(ctx, n.nsmgrCC)

	return n.newEndpoint(ctx, nse, generatorFunc, registryClient, additionalFunctionality...)
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

	ctx = log.Join(ctx, log.Empty())
	serve(ctx, u, ep.Register)

	nse.Url = u.String()

	// 3. Register with the node registry client
	var reg *registryapi.NetworkServiceEndpoint
	if reg, err = n.registerEndpoint(ctx, nse, registryClient); err != nil {
		return nil, err
	}

	nse.Name = reg.Name
	nse.ExpirationTime = reg.ExpirationTime

	log.FromContext(ctx).Infof("Started listen endpoint %s on %s.", nse.Name, u.String())

	return &EndpointEntry{Endpoint: ep, URL: u}, nil
}

// RegisterEndpoint registers endpoint on the node NSMgr with its own registry client
func (n *Node) RegisterEndpoint(ctx context.Context, nse *registryapi.NetworkServiceEndpoint) (*registryapi.NetworkServiceEndpoint, error) {
	registryClient, ok := n.nseRegistryClients.Load(nse.Name)
	if !ok {
		registryClient = registryclient.NewNetworkServiceEndpointRegistryClient(ctx, n.nsmgrCC)
	}
	return n.registerEndpoint(ctx, nse, registryClient)
}

func (n *Node) registerEndpoint(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	registryClient registryapi.NetworkServiceEndpointRegistryClient,
) (reg *registryapi.NetworkServiceEndpoint, err error) {
	if reg, err = registryClient.Register(ctx, nse); err != nil {
		return nil, err
	}

	n.nseRegistryClients.Store(reg.Name, registryClient)

	return reg, nil
}

// UnregisterForwarder unregisters forwarder from the node NSMgr with its own registry client
func (n *Node) UnregisterForwarder(ctx context.Context, nse *registryapi.NetworkServiceEndpoint) error {
	registryClient, ok := n.nseRegistryClients.Load(nse.Name)
	if !ok {
		registryClient = registryclient.NewNetworkServiceEndpointRegistryInterposeClient(ctx, n.nsmgrCC)
	}
	return n.unregisterEndpoint(ctx, nse, registryClient)
}

// UnregisterEndpoint unregisters endpoint from the node NSMgr with its own registry client
func (n *Node) UnregisterEndpoint(ctx context.Context, nse *registryapi.NetworkServiceEndpoint) error {
	registryClient, ok := n.nseRegistryClients.Load(nse.Name)
	if !ok {
		registryClient = registryclient.NewNetworkServiceEndpointRegistryClient(ctx, n.nsmgrCC)
	}
	return n.unregisterEndpoint(ctx, nse, registryClient)
}

func (n *Node) unregisterEndpoint(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	registryClient registryapi.NetworkServiceEndpointRegistryClient,
) error {
	_, err := registryClient.Unregister(ctx, nse)
	return err
}

// NewClient starts a new client and connects it to the node NSMgr
func (n *Node) NewClient(
	ctx context.Context,
	generatorFunc token.GeneratorFunc,
	additionalFunctionality ...networkservice.NetworkServiceClient,
) networkservice.NetworkServiceClient {
	ctx = log.Join(ctx, log.Empty())
	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(n.NSMgr.URL), DefaultDialOptions(generatorFunc)...)
	if err != nil {
		log.FromContext(ctx).Fatalf("Failed to dial node NSMgr: %s", err.Error())
	}

	go func() {
		defer func() { _ = cc.Close() }()
		<-ctx.Done()
	}()

	return client.NewClient(
		ctx,
		cc,
		client.WithAuthorizeClient(authorize.NewClient(authorize.Any())),
		client.WithAdditionalFunctionality(additionalFunctionality...),
	)
}
