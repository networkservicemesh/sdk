// Copyright (c) 2022 Cisco and/or its affiliates.
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

package injectspiffeid

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type injectSpiffeIDNSEServer struct {
	cert []byte
}

// NewNetworkServiceEndpointRegistryServer returns a server chain element putting spiffeID to context on Register and Unregister
func NewNetworkServiceEndpointRegistryServer(spiffeID string) registry.NetworkServiceEndpointRegistryServer {
	return &injectSpiffeIDNSEServer{
		cert: generateCert(spiffeID),
	}
}

func (s *injectSpiffeIDNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	ctx, err := withPeer(ctx, s.cert)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *injectSpiffeIDNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *injectSpiffeIDNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	ctx, err := withPeer(ctx, s.cert)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
