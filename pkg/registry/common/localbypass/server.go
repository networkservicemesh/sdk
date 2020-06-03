// Copyright (c) 2020 Cisco Systems, Inc.
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

// Package localbypass provides NetworkServiceRegistryServer that registers local Endpoints
// and adds them to localbypass.SocketMap
package localbypass

import (
	"context"
	"net/url"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

// LocalBypass - return interafce to return localbypass endpoints URIs
type LocalBypass interface {
	LoadEndpointURL(endpointName string) (*url.URL, bool)
}

// Registry - localbypass registry to find local registered endpoints, by it's ids.
type Registry interface {
	LocalBypass
	registry.NetworkServiceRegistryServer
}

type localBypassRegistry struct {
	// Map of names -> *url.URLs for local bypass to file sockets
	sockets sync.Map
	registry.NetworkServiceRegistryServer
}

// NewServer - creates a NetworkServiceRegistryServer that registers local Endpoints
//				and adds them to localbypass.SocketMap
func NewServer() Registry {
	return &localBypassRegistry{}
}

func (n *localBypassRegistry) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	u, err := url.Parse(request.GetNetworkServiceEndpoint().GetUrl())
	if err != nil {
		return nil, err
	}
	n.sockets.LoadOrStore(request.GetNetworkServiceEndpoint().GetName(), u)
	return next.NetworkServiceRegistryServer(ctx).RegisterNSE(ctx, request)
}

func (n *localBypassRegistry) BulkRegisterNSE(server registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	return next.NetworkServiceRegistryServer(server.Context()).BulkRegisterNSE(server)
}

func (n *localBypassRegistry) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	n.sockets.Delete(request.GetNetworkServiceEndpointName())
	return next.NetworkServiceRegistryServer(ctx).RemoveNSE(ctx, request)
}

func (n *localBypassRegistry) LoadEndpointURL(endpointName string) (*url.URL, bool) {
	if v, ok := n.sockets.Load(endpointName); ok && v != nil {
		if u, ok := v.(*url.URL); ok {
			return u, true
		}
	}
	return nil, false
}
