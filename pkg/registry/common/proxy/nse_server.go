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

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

type nseServer struct {
	proxyRegistryURL *url.URL
}

func (n *nseServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if !interdomain.Is(nse.Name) {
		return nse, nil
	}
	if n.proxyRegistryURL == nil {
		return nil, urlToProxyNotPassedErr
	}
	ctx = clienturlctx.WithClientURL(ctx, n.proxyRegistryURL)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (n nseServer) Find(q *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	if !isInterdomain(q.NetworkServiceEndpoint) {
		return nil
	}
	if n.proxyRegistryURL == nil {
		return urlToProxyNotPassedErr
	}
	ctx := clienturlctx.WithClientURL(s.Context(), n.proxyRegistryURL)
	return next.NetworkServiceEndpointRegistryServer(ctx).Find(q, streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, s))
}

func (n *nseServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if !interdomain.Is(nse.Name) {
		return new(empty.Empty), nil
	}
	if n.proxyRegistryURL == nil {
		return nil, urlToProxyNotPassedErr
	}
	ctx = clienturlctx.WithClientURL(ctx, n.proxyRegistryURL)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

// NewNetworkServiceEndpointRegistryServer creates new NetworkServiceEndpointRegistryServer that can proxying interdomain upstream to the remote registry by URL
func NewNetworkServiceEndpointRegistryServer(proxyRegistryURL *url.URL) registry.NetworkServiceEndpointRegistryServer {
	return &nseServer{
		proxyRegistryURL: proxyRegistryURL,
	}
}

func isInterdomain(nse *registry.NetworkServiceEndpoint) bool {
	if interdomain.Is(nse.Name) {
		return true
	}
	for _, ns := range nse.NetworkServiceNames {
		if interdomain.Is(ns) {
			return true
		}
	}
	return false
}
