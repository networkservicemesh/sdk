// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022 Cisco and/or its affiliates.
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

package checknse

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type checkNSEServer struct {
	*testing.T
	check func(*testing.T, *registry.NetworkServiceEndpoint)
}

// NewServer - returns NetworkServiceEndpointRegistryServer that checks the NSE passed in from the previous Server in the chain
//
//	t - *testing.T used for the check
//	check - function that checks the *registry.NetworkServiceEndpoint
func NewServer(t *testing.T, check func(*testing.T, *registry.NetworkServiceEndpoint)) registry.NetworkServiceEndpointRegistryServer {
	return &checkNSEServer{
		T:     t,
		check: check,
	}
}

func (s *checkNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	s.check(s.T, nse)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *checkNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	s.check(s.T, query.GetNetworkServiceEndpoint())
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *checkNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	s.check(s.T, nse)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
