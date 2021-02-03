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

// Package seturl implements a chain elemenet to set url to registered endpoints.
package seturl

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/endpointurls"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/stringurl"
)

type setURLNSEServer struct {
	url     string
	nses    *endpointurls.Map
	nseURLs stringurl.Map
}

// NewNetworkServiceEndpointRegistryServer creates new instance of NetworkServiceEndpointRegistryServer which set the passed NSMgr url
func NewNetworkServiceEndpointRegistryServer(u string, nses *endpointurls.Map) registry.NetworkServiceEndpointRegistryServer {
	return &setURLNSEServer{
		url:  u,
		nses: nses,
	}
}

func (s *setURLNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	u, err := url.Parse(nse.Url)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot register NSE with passed URL: %s", nse.Url)
	}
	if u.String() == "" {
		return nil, errors.Errorf("cannot register NSE with passed URL: %s", nse.Url)
	}

	nse.Url = s.url

	nse, err = next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}

	nse.Url = u.String()

	s.nseURLs.LoadOrStore(nse.Name, u)
	s.nses.LoadOrStore(*u, nse.Name)

	return nse, nil
}

func (s *setURLNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, s.findServer(server))
}

func (s *setURLNSEServer) findServer(server registry.NetworkServiceEndpointRegistry_FindServer) registry.NetworkServiceEndpointRegistry_FindServer {
	return &setURLNSEFindServer{
		nseURLs: &s.nseURLs,
		NetworkServiceEndpointRegistry_FindServer: server,
	}
}

func (s *setURLNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if u, ok := s.nseURLs.LoadAndDelete(nse.Name); ok {
		s.nses.Delete(*u)
	}

	nse.Url = s.url

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
