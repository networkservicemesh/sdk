// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

// Package localbypass implements a chain element to set NSMgr URL to endpoints on registration and set back endpoints
// URLs on find
package localbypass

import (
	"context"
	"net/url"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type localBypassNSEFindServer struct {
	*localBypassNSEServer
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (s *localBypassNSEFindServer) Send(nseResp *registry.NetworkServiceEndpointResponse) error {
	if u, ok := s.nseURLs.Load(nseResp.GetNetworkServiceEndpoint().GetName()); ok {
		nseResp.NetworkServiceEndpoint.Url = u.String()
	}

	if nseResp.GetNetworkServiceEndpoint().GetUrl() == s.nsmgrURL && !nseResp.GetDeleted() {
		return nil
	}

	return s.NetworkServiceEndpointRegistry_FindServer.Send(nseResp)
}

type localBypassNSEServer struct {
	nsmgrURL string
	nseURLs  genericsync.Map[string, *url.URL]
}

// NewNetworkServiceEndpointRegistryServer creates new instance of NetworkServiceEndpointRegistryServer which sets
// NSMgr URL to endpoints on registration and sets back endpoints URLs on find.
func NewNetworkServiceEndpointRegistryServer(nsmgrURL string) registry.NetworkServiceEndpointRegistryServer {
	return &localBypassNSEServer{
		nsmgrURL: nsmgrURL,
	}
}

func (s *localBypassNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (reg *registry.NetworkServiceEndpoint, err error) {
	u, loaded := s.nseURLs.Load(nse.GetName())
	if !loaded || u.String() != nse.GetUrl() {
		u, err = url.Parse(nse.GetUrl())
		if err != nil {
			return nil, errors.Wrapf(err, "cannot register NSE with passed URL: %s", nse.GetUrl())
		}
		if u.String() == "" {
			return nil, errors.Errorf("cannot register NSE with passed URL: %s", nse.GetUrl())
		}
		s.nseURLs.Store(nse.GetName(), u)
	}

	nse.Url = s.nsmgrURL

	reg, err = next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		if !loaded {
			s.nseURLs.Delete(nse.GetName())
		}
		return nil, err
	}

	reg.Url = u.String()
	s.nseURLs.Store(reg.GetName(), u)

	return reg, nil
}

func (s *localBypassNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, &localBypassNSEFindServer{
		localBypassNSEServer:                      s,
		NetworkServiceEndpointRegistry_FindServer: server,
	})
}

func (s *localBypassNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (_ *empty.Empty, err error) {
	if _, ok := s.nseURLs.Load(nse.GetName()); ok {
		nse.Url = s.nsmgrURL

		_, err = next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
		if err == nil {
			s.nseURLs.Delete(nse.GetName())
		}
	}
	return new(empty.Empty), err
}
