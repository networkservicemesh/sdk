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
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/common/interpose"
	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
	"github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/logger"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// Node is a NSMgr with resources
type Node struct {
	NSMgr     *NSMgrEntry
	log       logger.Logger
	resources []context.CancelFunc
}

func (n *Node) cleanup() {
	for _, r := range n.resources {
		r()
	}
}

// NewForwarder starts a new forwarder and registers it on the node NSMgr
func (n *Node) NewForwarder(
	ctx context.Context,
	nse *registry.NetworkServiceEndpoint,
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

	entry, err := n.newEndpoint(ctx, nse, generatorFunc, true, additionalFunctionality...)
	if err != nil {
		return nil, err
	}
	*ep = *entry

	return ep, nil
}

// NewEndpoint starts a new endpoint and registers it on the node NSMgr
func (n *Node) NewEndpoint(
	ctx context.Context,
	nse *registry.NetworkServiceEndpoint,
	generatorFunc token.GeneratorFunc,
	additionalFunctionality ...networkservice.NetworkServiceServer,
) (*EndpointEntry, error) {
	return n.newEndpoint(ctx, nse, generatorFunc, false, additionalFunctionality...)
}

func (n *Node) newEndpoint(
	ctx context.Context,
	nse *registry.NetworkServiceEndpoint,
	generatorFunc token.GeneratorFunc,
	isForwarder bool,
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

	serve(logger.WithLog(ctx, n.log), u, ep.Register)

	if nse.Url == "" {
		nse.Url = u.String()
	}

	// 3. Register to the node NSMgr
	cc := n.dialNSMgr(ctx)

	nsClient := registry.NewNetworkServiceRegistryClient(cc)
	for _, name := range nse.NetworkServiceNames {
		var service *registry.NetworkService
		service, err = nsClient.Register(ctx, &registry.NetworkService{
			Name:    name,
			Payload: "IP",
		})
		if err != nil {
			return nil, err
		}
		n.resources = append(n.resources, func() {
			_, _ = nsClient.Unregister(ctx, service)
		})
	}

	var interposeClient registry.NetworkServiceEndpointRegistryClient
	if isForwarder {
		interposeClient = interpose.NewNetworkServiceEndpointRegistryClient()
	} else {
		interposeClient = null.NewNetworkServiceEndpointRegistryClient()
	}

	nseClient := chain.NewNetworkServiceEndpointRegistryClient(
		refresh.NewNetworkServiceEndpointRegistryClient(
			refresh.WithChainContext(ctx)),
		interposeClient,
		registry.NewNetworkServiceEndpointRegistryClient(cc),
	)
	nse, err = nseClient.Register(ctx, nse)
	if err != nil {
		return nil, err
	}
	n.resources = append(n.resources, func() {
		_, _ = nseClient.Unregister(ctx, nse)
	})

	if isForwarder {
		n.log.Infof("Started listen forwarder %v on %v.", nse.Name, u.String())
	} else {
		n.log.Infof("Started listen endpoint %v on %v.", nse.Name, u.String())
	}

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
		fmt.Sprintf("nsc-%v", uuid.New().String()),
		nil,
		generatorFunc,
		n.dialNSMgr(ctx),
		additionalFunctionality...,
	)
}

func (n *Node) dialNSMgr(ctx context.Context) grpc.ClientConnInterface {
	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(n.NSMgr.URL),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		panic("failed to dial node NSMgr")
	}

	n.resources = append(n.resources, func() {
		_ = cc.Close()
	})

	return cc
}
