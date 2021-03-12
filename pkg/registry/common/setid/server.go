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

package setid

import (
	"context"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type setIDServer struct {
	names namesSet
}

// NewNetworkServiceEndpointRegistryServer creates new instance of NetworkServiceRegistryServer which set the unique
// name for the endpoint on registration
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return new(setIDServer)
}

func (s *setIDServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	name := nse.Name

	reg, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}

	if _, ok := s.names.Load(reg.Name); !ok && reg.Name == name {
		if reg.Name == "" {
			reg.Name = strings.Join(reg.NetworkServiceNames, "-")
		}
		reg.Name = uuid.New().String() + "-" + reg.Name
	}

	s.names.Store(reg.Name, struct{}{})

	return reg, nil
}

func (s *setIDServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *setIDServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	s.names.Delete(nse.Name)

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

var _ registry.NetworkServiceEndpointRegistryServer = &setIDServer{}
