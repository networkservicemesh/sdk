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
	"net/url"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// Node is a NSMgr with Forwarder, NSE registry clients
type Node struct {
	ctx                     context.Context
	NSMgr                   *NSMgrEntry
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
		connect.NewServer(ctx,
			client.NewCrossConnectClientFactory(
				nse.Name,
				// What to call onHeal
				addressof.NetworkServiceClient(adapters.NewServerToClient(ep)),
				generatorFunc),
			grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		))

	entry, err := n.newEndpoint(ctx, nse, generatorFunc, n.ForwarderRegistryClient, additionalFunctionality...)
	if err != nil {
		return nil, err
	}
	*ep = *entry

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
	ep := endpoint.NewServer(
		ctx,
		nse.Name,
		authorize.NewServer(),
		generatorFunc,
		additionalFunctionality...,
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

	for _, nsName := range nse.NetworkServiceNames {
		if _, err = n.NSRegistryClient.Register(ctx, &registryapi.NetworkService{
			Name:    nsName,
			Payload: payload.IP,
		}); err != nil {
			return nil, err
		}
	}

	var reg *registryapi.NetworkServiceEndpoint
	if reg, err = registryClient.Register(ctx, nse); err != nil {
		return nil, err
	}

	nse.Name = reg.Name
	nse.ExpirationTime = reg.ExpirationTime

	log.FromContext(ctx).Infof("Started listen endpoint %s on %s.", nse.Name, u.String())

	return &EndpointEntry{Endpoint: ep, URL: u}, nil
}

// NewClient starts a new client and connects it to the node NSMgr
func (n *Node) NewClient(
	ctx context.Context,
	generatorFunc token.GeneratorFunc,
	additionalFunctionality ...networkservice.NetworkServiceClient,
) networkservice.NetworkServiceClient {
	ctx = log.Join(ctx, log.Empty())
	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(n.NSMgr.URL),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		log.FromContext(ctx).Fatalf("Failed to dial node NSMgr: %s", err.Error())
	}

	go func() {
		defer func() { _ = cc.Close() }()
		<-ctx.Done()
	}()

	return client.NewClient(
		ctx,
		fmt.Sprintf("nsc-%v", uuid.New().String()),
		nil,
		generatorFunc,
		cc,
		additionalFunctionality...,
	)
}
