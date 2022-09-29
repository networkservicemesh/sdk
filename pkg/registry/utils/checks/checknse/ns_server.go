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

type checkNSServer struct {
	*testing.T
	check func(*testing.T, *registry.NetworkService)
}

// NewNetworkServiceRegistryServer - returns NetworkServiceRegistryServer that checks the NS passed in from the previous Server in the chain
//
//	t - *testing.T used for the check
//	check - function that checks the *registry.NetworkService
func NewNetworkServiceRegistryServer(t *testing.T, check func(*testing.T, *registry.NetworkService)) registry.NetworkServiceRegistryServer {
	return &checkNSServer{
		T:     t,
		check: check,
	}
}

func (s *checkNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	s.check(s.T, ns)
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
}

func (s *checkNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	s.check(s.T, query.NetworkService)
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (s *checkNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	s.check(s.T, ns)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}
