// Copyright (c) 2021-2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

//go:build linux
// +build linux

package sendfd

import (
	"context"

	"github.com/edwarnicke/grpcfd"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type sendfdNSEServer struct{}

func (n *sendfdNSEServer) Register(ctx context.Context, service *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, service)
}

func (n *sendfdNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	var nextServer = server

	sender, ok := grpcfd.FromContext(server.Context())
	if ok {
		nextServer = &sendfdNSEFindServer{
			NetworkServiceEndpointRegistry_FindServer: server,
			transceiver: sender,
		}
	}

	return next.NetworkServiceEndpointRegistryServer(nextServer.Context()).Find(query, nextServer)
}

func (n *sendfdNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

// NewNetworkServiceEndpointRegistryServer - returns a new null server that does nothing but call next.NetworkServiceEndpointRegistryServer(ctx).
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return new(sendfdNSEServer)
}

type sendfdNSEFindServer struct {
	transceiver grpcfd.FDTransceiver
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (s *sendfdNSEFindServer) Send(nseResp *registry.NetworkServiceEndpointResponse) error {
	var inodeURLToFileURLMap = make(map[string]string)
	if err := sendFDAndSwapFileToInode(s.transceiver, nseResp.GetNetworkServiceEndpoint(), inodeURLToFileURLMap); err != nil {
		return err
	}
	return s.NetworkServiceEndpointRegistry_FindServer.Send(nseResp)
}
