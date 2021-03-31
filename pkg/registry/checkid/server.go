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

// Package checkid provides NSE server chain element for checking for nse.Name duplicates
package checkid

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type setIDServer struct {
	namesUrls namesUrlsMap
}

// NewNetworkServiceEndpointRegistryServer creates a new NSE server chain element checking for nse.Name collisions
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return new(setIDServer)
}

func (s *setIDServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (reg *registry.NetworkServiceEndpoint, err error) {
	if nse.Name == "" || nse.Url == "" {
		return nil, errors.Errorf("nse.Name and nse.Url should be not empty: %v", nse)
	}

	u, loaded := s.namesUrls.Load(nse.Name)
	if loaded && u != nse.Url {
		return nil, &DuplicateError{
			name:     nse.Name,
			expected: u,
			actual:   nse.Url,
		}
	}

	name, u := nse.Name, nse.Url

	reg, err = next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}

	if !loaded {
		s.namesUrls.Store(name, u)
	}

	return reg, err
}

func (s *setIDServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *setIDServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if _, ok := s.namesUrls.LoadAndDelete(nse.Name); !ok {
		return new(empty.Empty), nil
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
