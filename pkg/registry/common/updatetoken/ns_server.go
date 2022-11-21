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

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type updateTokenNSServer struct {
	tokenGenerator token.GeneratorFunc
}

// NewNetworkServiceRegistryServer - creates a NetworkServiceRegistryServer chain element to update NetworkService.Path token information
func NewNetworkServiceRegistryServer(tokenGenerator token.GeneratorFunc) registry.NetworkServiceRegistryServer {
	return &updateTokenNSServer{
		tokenGenerator: tokenGenerator,
	}
}

func (s *updateTokenNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	path, err := grpcmetadata.PathFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if prev := path.GetPrevPathSegment(); prev != nil {
		tok, expireTime, tokenErr := token.FromContext(ctx)

		if tokenErr != nil {
			log.FromContext(ctx).Warnf("an error during getting token from the context: %+v", tokenErr)
		} else {
			expires := expireTime.Local()
			prev.Expires = expires
			prev.Token = tok
			id, idErr := getIDFromToken(tok)
			if idErr != nil {
				return nil, idErr
			}
			ns.PathIds = updatePathIds(ns.PathIds, int(path.Index-1), id.String())
		}
	}
	err = updateToken(ctx, path, s.tokenGenerator)
	if err != nil {
		return nil, err
	}

	id, err := getIDFromToken(path.PathSegments[path.Index].Token)
	if err != nil {
		return nil, err
	}
	ns.PathIds = updatePathIds(ns.PathIds, int(path.Index), id.String())

	return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
}

func (s *updateTokenNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

// TODO: Finish this method. Append spiffeID of PathSegment to nse.PathIds
func (s *updateTokenNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	path, err := grpcmetadata.PathFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if prev := path.GetPrevPathSegment(); prev != nil {
		tok, expireTime, tokenErr := token.FromContext(ctx)

		if tokenErr != nil {
			log.FromContext(ctx).Warnf("an error during getting token from the context: %+v", tokenErr)
		} else {
			expires := expireTime.Local()
			prev.Expires = expires
			prev.Token = tok

			id, idErr := getIDFromToken(tok)
			if idErr != nil {
				return nil, idErr
			}
			ns.PathIds = updatePathIds(ns.PathIds, int(path.Index-1), id.String())
		}
	}
	err = updateToken(ctx, path, s.tokenGenerator)
	if err != nil {
		return nil, err
	}

	id, err := getIDFromToken(path.PathSegments[path.Index].Token)
	if err != nil {
		return nil, err
	}
	ns.PathIds = updatePathIds(ns.PathIds, int(path.Index), id.String())

	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}
