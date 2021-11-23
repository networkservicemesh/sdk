// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package localcache

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type tail struct{}

func (t *tail) Register(_ context.Context, r *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return r, nil
}

func (t *tail) Find(_ *registry.NetworkServiceEndpointQuery, _ registry.NetworkServiceEndpointRegistry_FindServer) error {
	return nil
}

func (t *tail) Unregister(context.Context, *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return new(empty.Empty), nil
}

var _ registry.NetworkServiceEndpointRegistryServer = &tail{}

type localcacheNSEServer struct {
	cache registry.NetworkServiceEndpointRegistryServer
}

// NewNetworkServiceEndpointRegistryServer - returns a new chain element that stores registered nses even if remote registry is unavailable
func NewNetworkServiceEndpointRegistryServer(cache registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
	c := next.NewNetworkServiceEndpointRegistryServer(cache, &tail{})
	return &localcacheNSEServer{
		cache: c,
	}
}

func (n *localcacheNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	r, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return n.cache.Register(ctx, nse)
	}
	return n.cache.Register(ctx, r)
}

func (n *localcacheNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	_ = next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
	return n.cache.Find(query, server)
}

func (n *localcacheNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	ret, err := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
	retC, errC := n.cache.Unregister(ctx, nse)
	if err != nil && errC != nil {
		return ret, errors.Wrap(err, errC.Error())
	}
	if errC != nil {
		return retC, errC
	}
	return ret, err
}
