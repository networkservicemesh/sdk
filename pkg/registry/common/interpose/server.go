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

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/stringurl"
)

type interposeRegistryServer struct {
	endpoints *stringurl.Map
}

// NewNetworkServiceRegistryServer - creates a NetworkServiceRegistryServer that registers local Cross connect Endpoints
//				and adds them to Map
func NewNetworkServiceRegistryServer(nses *stringurl.Map) registry.NetworkServiceEndpointRegistryServer {
	return &interposeRegistryServer{endpoints: nses}
}

func (rs *interposeRegistryServer) Register(ctx context.Context, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if !Is(request.Name) {
		return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, request)
	}

	u, err := url.Parse(request.Url)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot register cross NSE with passed URL: %s", request.Url)
	}
	if u.String() == "" {
		return nil, errors.Errorf("cannot register cross NSE with passed URL: %s", request.Url)
	}

	rs.endpoints.LoadOrStore(request.Name, u)

	return request, nil
}

func (rs *interposeRegistryServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	// No need to modify find logic.
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, s)
}

func (rs *interposeRegistryServer) Unregister(ctx context.Context, request *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if !Is(request.Name) {
		return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, request)
	}

	rs.endpoints.Delete(request.Name)

	return new(empty.Empty), nil
}

var _ registry.NetworkServiceEndpointRegistryServer = (*interposeRegistryServer)(nil)
