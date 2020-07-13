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

package proxy

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

type nsServer struct {
	proxyRegistryURL *url.URL
}

func (n *nsServer) Register(ctx context.Context, nse *registry.NetworkService) (*registry.NetworkService, error) {
	if !interdomain.Is(nse.Name) {
		return nse, nil
	}
	if n.proxyRegistryURL == nil {
		return nil, urlToProxyNotPassedErr
	}
	ctx = clienturl.WithClientURL(ctx, n.proxyRegistryURL)
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, nse)
}

func (n nsServer) Find(q *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	if !interdomain.Is(q.NetworkService.Name) {
		return nil
	}
	if n.proxyRegistryURL == nil {
		return urlToProxyNotPassedErr
	}
	ctx := clienturl.WithClientURL(s.Context(), n.proxyRegistryURL)
	return next.NetworkServiceRegistryServer(ctx).Find(q, streamcontext.NetworkServiceRegistryFindServer(ctx, s))
}

func (n *nsServer) Unregister(ctx context.Context, nse *registry.NetworkService) (*empty.Empty, error) {
	if !interdomain.Is(nse.Name) {
		return new(empty.Empty), nil
	}
	if n.proxyRegistryURL == nil {
		return nil, urlToProxyNotPassedErr
	}
	ctx = clienturl.WithClientURL(ctx, n.proxyRegistryURL)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, nse)
}

// NewNetworkServiceRegistryServer creates new NetworkServiceRegistryServer that can proxying interdomain upstream to the remote registry by URL
func NewNetworkServiceRegistryServer(proxyRegistryURL *url.URL) registry.NetworkServiceRegistryServer {
	return &nsServer{
		proxyRegistryURL: proxyRegistryURL,
	}
}
