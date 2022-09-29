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

package updatetoken

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type updateTokenNSEServer struct {
	tokenGenerator token.GeneratorFunc
}

// NewNetworkServiceEndpointRegistryServer - creates a NetworkServiceEndpointRegistryServer chain element to update NetworkServiceEndpoint.Path token information
func NewNetworkServiceEndpointRegistryServer(tokenGenerator token.GeneratorFunc) registry.NetworkServiceEndpointRegistryServer {
	return &updateTokenNSEServer{
		tokenGenerator: tokenGenerator,
	}
}

func (s *updateTokenNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if prev := GetPrevPathSegment(nse.GetPath()); prev != nil {
		var token, expireTime, err = token.FromContext(ctx)

		if err != nil {
			log.FromContext(ctx).Warnf("an error during getting token from the context: %+v", err)
		} else {
			expires := timestamppb.New(expireTime.Local())
			prev.Expires = expires
			prev.Token = token
		}
	}
	err := updateToken(ctx, nse.GetPath(), s.tokenGenerator)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *updateTokenNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *updateTokenNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if prev := GetPrevPathSegment(nse.GetPath()); prev != nil {
		var token, expireTime, err = token.FromContext(ctx)

		if err != nil {
			log.FromContext(ctx).Warnf("an error during getting token from the context: %+v", err)
		} else {
			expires := timestamppb.New(expireTime.Local())

			prev.Expires = expires
			prev.Token = token
		}
	}
	err := updateToken(ctx, nse.GetPath(), s.tokenGenerator)
	if err != nil {
		return nil, err
	}

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
