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
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

type nseSwapRegistryServer struct {
	domain        string
	proxyNSMgrURL *url.URL
}

// NewNetworkServiceEndpointRegistryServer creates new NetworkServiceEndpointRegistryServer which can set for outgoing network service endpoint name to interdomain name and can set URL to interdomain URL.
// Also updates URL and Name of incoming NSE for proxy network service manager.
func NewNetworkServiceEndpointRegistryServer(domain string, proxyNSMgrURL *url.URL) registry.NetworkServiceEndpointRegistryServer {
	return &nseSwapRegistryServer{
		domain:        domain,
		proxyNSMgrURL: proxyNSMgrURL,
	}
}

func (n *nseSwapRegistryServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	nse.Name = interdomain.Join(interdomain.Target(nse.Name), n.domain)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

type findNSESwapServer struct {
	proxyNSMgrURL *url.URL
	localDomain   string
	remoteDomain  string
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (s *findNSESwapServer) Send(nse *registry.NetworkServiceEndpoint) error {
	if s.remoteDomain == s.localDomain {
		nse.Name = interdomain.Target(nse.Name)
	} else {
		nse.Name = interdomain.Join(interdomain.Target(nse.Name), nse.Url)
		nse.Url = s.proxyNSMgrURL.String()
	}
	for i, service := range nse.NetworkServiceNames {
		if domain := interdomain.Domain(service); domain == s.remoteDomain {
			nse.NetworkServiceNames[i] = interdomain.Target(service)
		}
	}
	return s.NetworkServiceEndpointRegistry_FindServer.Send(nse)
}

func (n *nseSwapRegistryServer) Find(q *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	remoteDomain := extractDomain(q.NetworkServiceEndpoint)
	q.NetworkServiceEndpoint.Name = interdomain.Target(q.NetworkServiceEndpoint.Name)
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(
		q,
		&findNSESwapServer{
			NetworkServiceEndpointRegistry_FindServer: s,
			proxyNSMgrURL: n.proxyNSMgrURL,
			localDomain:   n.domain,
			remoteDomain:  remoteDomain,
		})
}

func (n *nseSwapRegistryServer) Unregister(ctx context.Context, ns *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	ns.Name = interdomain.Join(interdomain.Target(ns.Name), n.domain)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, ns)
}

func extractDomain(nse *registry.NetworkServiceEndpoint) string {
	domain := interdomain.Domain(nse.Name)
	if domain != "" {
		nse.Name = interdomain.Target(nse.Name)
		return domain
	}
	for i, service := range nse.NetworkServiceNames {
		if domain = interdomain.Domain(service); domain != "" {
			nse.NetworkServiceNames[i] = interdomain.Target(service)
			return domain
		}
	}
	return ""
}

var _ registry.NetworkServiceEndpointRegistryServer = (*nseSwapRegistryServer)(nil)
