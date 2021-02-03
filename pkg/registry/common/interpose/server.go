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

	"github.com/networkservicemesh/sdk/pkg/registry/common/endpointurls"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/stringurl"
)

type interposeRegistryServer struct {
	interposeNSEs *endpointurls.Map
	interposeURLs stringurl.Map
}

// NewNetworkServiceEndpointRegistryServer - creates a NetworkServiceRegistryServer that registers local Cross connect Endpoints
//				and adds them to Map
func NewNetworkServiceEndpointRegistryServer(interposeNSEs *endpointurls.Map) registry.NetworkServiceEndpointRegistryServer {
	return &interposeRegistryServer{
		interposeNSEs: interposeNSEs,
	}
}

func (s *interposeRegistryServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if !isInterposeName(nse.Name) {
		return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	}

	u, err := url.Parse(nse.Url)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot register cross NSE with passed URL: %s", nse.Url)
	}
	if u.String() == "" {
		return nil, errors.Errorf("cannot register cross NSE with passed URL: %s", nse.Url)
	}

	s.interposeURLs.LoadOrStore(nse.Name, u)
	s.interposeNSEs.LoadOrStore(*u, nse.Name)

	return nse, nil
}

func (s *interposeRegistryServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	// No need to modify find logic.
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *interposeRegistryServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if !isInterposeName(nse.Name) {
		return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
	}

	if u, ok := s.interposeURLs.LoadAndDelete(nse.Name); ok {
		s.interposeNSEs.Delete(*u)
	}

	return new(empty.Empty), nil
}

var _ registry.NetworkServiceEndpointRegistryServer = (*interposeRegistryServer)(nil)
