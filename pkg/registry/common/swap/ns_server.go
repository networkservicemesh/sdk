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

package swap

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

type nsSwapRegistryServer struct {
	domain string
}

// NewNetworkServiceRegistryServer creates new NetworkServiceRegistry which can set for outgoing network service name to interdomain name
func NewNetworkServiceRegistryServer(domain string) registry.NetworkServiceRegistryServer {
	return &nsSwapRegistryServer{
		domain: domain,
	}
}

func (n *nsSwapRegistryServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	ns.Name = interdomain.Join(interdomain.Target(ns.Name), n.domain)
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
}

func (n *nsSwapRegistryServer) Find(q *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	return next.NetworkServiceRegistryServer(s.Context()).Find(q, s)
}

func (n *nsSwapRegistryServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	ns.Name = interdomain.Join(interdomain.Target(ns.Name), n.domain)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}

var _ registry.NetworkServiceRegistryServer = (*nsSwapRegistryServer)(nil)
