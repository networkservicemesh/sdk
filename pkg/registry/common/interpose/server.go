// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package interpose provides NetworkServiceRegistryServer that registers local Endpoints
// and adds them to Map
package interpose

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/stringurl"

	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

// interposeNSEName - a common prefix for all registered cross NSEs
const interposeNSEName = "interpose-nse#"

type interposeRegistry struct {
	endpoints *stringurl.Map
}

func (l *interposeRegistry) Register(ctx context.Context, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if strings.HasPrefix(request.Name, interposeNSEName) {
		endpointURL, err := url.Parse(request.Url)
		if err != nil {
			return nil, err
		}
		if endpointURL == nil {
			return nil, errors.New("invalid endpoint URL passed with context")
		}

		// Generate uniq name for interpose endpoint, since we use it in maps.
		request.Name = interposeNSEName + uuid.New().String()

		u, err := url.Parse(request.Url)
		if err != nil {
			return nil, errors.Wrapf(err, "Cannot register cross nse with passed URL")
		}
		l.endpoints.LoadOrStore(request.Name, u)
		return request, nil
	}

	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, request)
}

func (l *interposeRegistry) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	// No need to modify find logic.
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, s)
}

func (l *interposeRegistry) Unregister(ctx context.Context, request *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if strings.HasPrefix(request.Name, interposeNSEName) {
		l.endpoints.Delete(request.Name)
		return &empty.Empty{}, nil
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, request)
}

// NewNetworkServiceRegistryServer - creates a NetworkServiceRegistryServer that registers local Cross connect Endpoints
//				and adds them to Map
func NewNetworkServiceRegistryServer(nses *stringurl.Map) registry.NetworkServiceEndpointRegistryServer {
	return &interposeRegistry{endpoints: nses}
}
