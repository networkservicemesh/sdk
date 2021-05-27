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

package clienturl

import (
	"context"
	"net/url"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
)

type clientURLNSServer struct {
	u *url.URL
}

// NewNetworkServiceRegistryServer - returns a new NS registry server chain element that sets client URL in context
func NewNetworkServiceRegistryServer(u *url.URL) registry.NetworkServiceRegistryServer {
	return &clientURLNSServer{
		u: u,
	}
}

func (s *clientURLNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	ctx = clienturlctx.WithClientURL(ctx, s.u)
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
}

func (s *clientURLNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	server = streamcontext.NetworkServiceRegistryFindServer(clienturlctx.WithClientURL(server.Context(), s.u), server)
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (s *clientURLNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*emptypb.Empty, error) {
	ctx = clienturlctx.WithClientURL(ctx, s.u)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}
