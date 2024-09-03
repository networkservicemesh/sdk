// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Package swapip allows to replace internal NSE address to external for register/unregister/find queries.
package swapip

import (
	"context"
	"net"
	"net/url"
	"sync/atomic"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type swapIPFindNSEServer struct {
	registry.NetworkServiceEndpointRegistry_FindServer
	m   map[string]string
	ctx context.Context
}

func (s *swapIPFindNSEServer) Send(nseResp *registry.NetworkServiceEndpointResponse) error {
	trySwapIP(s.ctx, nseResp.GetNetworkServiceEndpoint(), s.m)
	if err := s.NetworkServiceEndpointRegistry_FindServer.Send(nseResp); err != nil {
		return errors.Wrapf(err, "NetworkServiceEndpointRegistry find server failed to send a response %s", nseResp.String())
	}
	return nil
}

type swapIPNSEServer struct {
	swapIPMap *atomic.Value
}

func (n *swapIPNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	m := n.swapIPMap.Load().(map[string]string)
	trySwapIP(ctx, nse, m)
	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err == nil {
		trySwapIP(ctx, resp, m)
	}
	return resp, err
}

func (n *swapIPNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	m := n.swapIPMap.Load().(map[string]string)
	trySwapIP(server.Context(), query.GetNetworkServiceEndpoint(), m)
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, &swapIPFindNSEServer{NetworkServiceEndpointRegistry_FindServer: server, m: m, ctx: server.Context()})
}

func (n *swapIPNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	m := n.swapIPMap.Load().(map[string]string)
	trySwapIP(ctx, nse, m)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

func trySwapIP(ctx context.Context, nse *registry.NetworkServiceEndpoint, ipMap map[string]string) {
	logger := log.FromContext(ctx)

	u, err := url.Parse(nse.GetUrl())
	defer func() {
		if err != nil {
			logger.Debugf("can not parse incomming url: %v, err: %v", nse.GetUrl(), err)
		}
	}()

	if err != nil {
		return
	}

	h, p, err := net.SplitHostPort(u.Host)
	if err != nil {
		return
	}

	if v, ok := ipMap[h]; ok {
		logger.Debugf("swapping %v to %v", h, v)
		u.Host = net.JoinHostPort(v, p)
		nse.Url = u.String()
	}
}

// NewNetworkServiceEndpointRegistryServer creates a new seturl registry.NetworkServiceEndpointRegistryServer.
func NewNetworkServiceEndpointRegistryServer(updateMapCh <-chan map[string]string) registry.NetworkServiceEndpointRegistryServer {
	v := new(atomic.Value)
	v.Store(map[string]string{})

	go func() {
		for m := range updateMapCh {
			v.Store(m)
		}
	}()

	return &swapIPNSEServer{
		swapIPMap: v,
	}
}
