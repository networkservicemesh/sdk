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

type clientURLNSEServer struct {
	u *url.URL
}

// NewNetworkServiceEndpointRegistryServer - returns a new NSE registry server chain element that sets client URL in context
func NewNetworkServiceEndpointRegistryServer(u *url.URL) registry.NetworkServiceEndpointRegistryServer {
	return &clientURLNSEServer{
		u: u,
	}
}

func (s *clientURLNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	ctx = clienturlctx.WithClientURL(ctx, s.u)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *clientURLNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	server = streamcontext.NetworkServiceEndpointRegistryFindServer(clienturlctx.WithClientURL(server.Context(), s.u), server)
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *clientURLNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*emptypb.Empty, error) {
	ctx = clienturlctx.WithClientURL(ctx, s.u)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
