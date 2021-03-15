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

// Package localbypass implements a chain element to set NSMgr URL to endpoints on registration and set back endpoints
// URLs on find
package localbypass

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/stringurl"
)

type localBypassNSEServer struct {
	nsmgrURL string
	nseURLs  stringurl.Map
}

// NewNetworkServiceEndpointRegistryServer creates new instance of NetworkServiceEndpointRegistryServer which sets
// NSMgr URL to endpoints on registration and sets back endpoints URLs on find
func NewNetworkServiceEndpointRegistryServer(nsmgrURL string) registry.NetworkServiceEndpointRegistryServer {
	return &localBypassNSEServer{
		nsmgrURL: nsmgrURL,
	}
}

// TODO: this is totally incorrect if there are concurrent events for the same nse.Name, think about adding
//       serialize into the registry chain
func (s *localBypassNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	logger := log.FromContext(ctx).WithField("localBypassNSEServer", "Register")

	u, err := url.Parse(nse.Url)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot register NSE with passed URL: %s", nse.Url)
	}
	if u.String() == "" {
		return nil, errors.Errorf("cannot register NSE with passed URL: %s", nse.Url)
	}

	nse.Url = s.nsmgrURL

	// We cannot store `nse.Name` -> `nse.Url` mapping at the moment, because we will have `nse.Name` only after
	// `next.Register` call.
	reg, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}

	name := reg.Name
	s.nseURLs.Store(name, u)

	// Now we already have nse registered and so we have `reg.Name` -> `nse.Url` mapping stored, but we possibly may
	// have filtered out Register update event in `localBypassNSEFindServer.Send`, so we need to refresh the
	// registration to produce such update event again.
	reg, err = next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, reg)
	if err != nil {
		if _, unregisterErr := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, reg); unregisterErr != nil {
			logger.Errorf("failed to unregister endpoint on error: %s", unregisterErr.Error())
		}
		return nil, errors.Wrapf(err, "failed to refresh endpoint registration: %s", name)
	}

	reg.Url = u.String()

	return reg, nil
}

func (s *localBypassNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, &localBypassNSEFindServer{
		localBypassNSEServer:                      s,
		NetworkServiceEndpointRegistry_FindServer: server,
	})
}

func (s *localBypassNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	nse.Url = s.nsmgrURL

	_, err := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)

	s.nseURLs.Delete(nse.Name)

	return new(empty.Empty), err
}
