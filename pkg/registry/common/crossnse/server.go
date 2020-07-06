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

// Package crossnse provides NetworkServiceRegistryServer that registers local Endpoints
// and adds them to Map
package crossnse

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

// CrossNSEName - a common prefix for all registered cross NSEs
const CrossNSEName = "cross-connect-nse#"

// Map - interface for map from networkServiceEndpoint names to cross connect NSE registrations
type Map interface {
	LoadOrStore(name string, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, bool)
	Delete(name string)
}

type localBypassRegistry struct {
	nses Map
}

func (l *localBypassRegistry) Register(ctx context.Context, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if strings.HasSuffix(request.Name, CrossNSEName) {
		endpointURL, err := url.Parse(request.Url)
		if err != nil {
			return nil, err
		}
		if endpointURL == nil {
			return nil, errors.New("invalid endpoint URL passed with context")
		}

		if request.Name == CrossNSEName {
			// Generate uniq name only if full equal to nses prefix.
			request.Name = CrossNSEName + uuid.New().String()
		}
		l.nses.LoadOrStore(request.Name, request)
		return request, nil
	}

	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, request)
}

func (l *localBypassRegistry) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	// No need to modify find logic.
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, s)
}

func (l *localBypassRegistry) Unregister(ctx context.Context, request *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if strings.HasSuffix(request.Name, CrossNSEName) {
		l.nses.Delete(request.Name)
		return &empty.Empty{}, nil
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, request)
}

// NewNetworkServiceRegistryServer - creates a NetworkServiceRegistryServer that registers local Cross connect Endpoints
//				and adds them to Map
func NewNetworkServiceRegistryServer(nses Map) registry.NetworkServiceEndpointRegistryServer {
	return &localBypassRegistry{nses: nses}
}
