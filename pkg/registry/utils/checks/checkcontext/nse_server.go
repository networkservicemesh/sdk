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

package checkcontext

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

// NewNSEServer - returns NetworkServiceEndpointRegistryServer that checks the context passed in from the previous Server in the chain
//             t - *testing.T used for the check
//             check - function that checks the context.Context
func NewNSEServer(t *testing.T, check func(*testing.T, context.Context)) registry.NetworkServiceEndpointRegistryServer {
	return &checkContextNSEServer{
		T:     t,
		check: check,
	}
}

type checkContextNSEServer struct {
	*testing.T
	check func(*testing.T, context.Context)
}

func (s *checkContextNSEServer) Register(ctx context.Context, service *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	s.check(s.T, ctx)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, service)
}

func (s *checkContextNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	s.check(s.T, server.Context())
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *checkContextNSEServer) Unregister(ctx context.Context, service *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	s.check(s.T, ctx)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, service)
}
