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

package interpose

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type interposeRegistryServer struct {
	backend registry.NetworkServiceEndpointRegistryServer
}

// NewNetworkServiceEndpointRegistryServer - creates a NetworkServiceRegistryServer that registers local Cross connect Endpoints
//				and adds them to Map
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return &interposeRegistryServer{
		backend: chain.NewNetworkServiceEndpointRegistryServer(
			memory.NewNetworkServiceEndpointRegistryServer(),
			new(breakNSEServer),
		),
	}
}

func (s *interposeRegistryServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if !Is(nse.Name) {
		return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	}

	u, err := url.Parse(nse.Url)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot register cross NSE with passed URL: %s", nse.Url)
	}
	if u.String() == "" {
		return nil, errors.Errorf("cannot register cross NSE with passed URL: %s", nse.Url)
	}

	return s.backend.Register(ctx, nse)
}

func (s *interposeRegistryServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	if Is(query.NetworkServiceEndpoint.GetName()) {
		query.NetworkServiceEndpoint.NetworkServiceNames = nil
		if err := s.backend.Find(query, server); err != nil {
			return err
		}
		return nil
	}
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *interposeRegistryServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if !Is(nse.Name) {
		return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
	}

	return s.backend.Unregister(ctx, nse)
}

var _ registry.NetworkServiceEndpointRegistryServer = (*interposeRegistryServer)(nil)

// TODO Should we use it? Should we move it to separate pkg and use in the next pkg?
type breakNSEServer struct{}

func (*breakNSEServer) Register(_ context.Context, r *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return r, nil
}

func (*breakNSEServer) Find(_ *registry.NetworkServiceEndpointQuery, _ registry.NetworkServiceEndpointRegistry_FindServer) error {
	return nil
}

func (*breakNSEServer) Unregister(context.Context, *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return new(empty.Empty), nil
}

var _ registry.NetworkServiceEndpointRegistryServer = &breakNSEServer{}
