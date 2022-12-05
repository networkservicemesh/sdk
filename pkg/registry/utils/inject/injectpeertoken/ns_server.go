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

package injectpeertoken

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type injectSpiffeIDNSServer struct {
	peerToken string
}

// NewNetworkServiceRegistryServer returns a server chain element putting spiffeID to context on Register and Unregister
func NewNetworkServiceRegistryServer(peerToken string) registry.NetworkServiceRegistryServer {
	return &injectSpiffeIDNSServer{
		peerToken: peerToken,
	}
}

func (s *injectSpiffeIDNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	ctx = withPeerToken(ctx, s.peerToken)
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
}

func (s *injectSpiffeIDNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (s *injectSpiffeIDNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*emptypb.Empty, error) {
	ctx = withPeerToken(ctx, s.peerToken)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}
