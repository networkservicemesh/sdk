// Copyright (c) 2022-2024 Cisco and/or its affiliates.
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

package updatepath

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type updatePathNSEServer struct {
	tokenGenerator token.GeneratorFunc
}

// NewNetworkServiceEndpointRegistryServer - creates a new updatePath server to update NetworkServiceEndpoint path.
func NewNetworkServiceEndpointRegistryServer(tokenGenerator token.GeneratorFunc) registry.NetworkServiceEndpointRegistryServer {
	return &updatePathNSEServer{
		tokenGenerator: tokenGenerator,
	}
}

func (s *updatePathNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	path := grpcmetadata.PathFromContext(ctx)

	// Update path
	peerTok, _, peerTokenErr := token.FromContext(ctx)
	if peerTokenErr != nil {
		log.FromContext(ctx).Warnf("an error during getting peer token from the context: %+v", peerTokenErr)
	}
	tok, expirationTime, tokenErr := generateToken(ctx, s.tokenGenerator)
	if tokenErr != nil {
		return nil, errors.Wrap(tokenErr, "an error during generating token")
	}
	path, index, err := updatePath(path, peerTok, tok)
	if err != nil {
		return nil, err
	}

	// Update path ids
	peerID, idErr := getIDFromToken(peerTok)
	if idErr != nil {
		log.FromContext(ctx).Warnf("an error during parsing peer token: %+v", tokenErr)
	}
	id, idErr := getIDFromToken(tok)
	if idErr != nil {
		return nil, idErr
	}
	nse.PathIds = updatePathIds(nse.GetPathIds(), int(path.Index-1), peerID.String())
	nse.PathIds = updatePathIds(nse.GetPathIds(), int(path.Index), id.String())

	ctx = withExpirationTime(ctx, &expirationTime)

	nse, err = next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}
	path.Index = index
	return nse, nil
}

func (s *updatePathNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *updatePathNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	path := grpcmetadata.PathFromContext(ctx)

	// Update path
	peerTok, _, tokenErr := token.FromContext(ctx)
	if tokenErr != nil {
		log.FromContext(ctx).Warnf("an error during getting peer token from the context: %+v", tokenErr)
	}
	tok, _, tokenErr := generateToken(ctx, s.tokenGenerator)
	if tokenErr != nil {
		return nil, errors.Wrap(tokenErr, "an error during generating token")
	}
	path, index, err := updatePath(path, peerTok, tok)
	if err != nil {
		return nil, err
	}

	// Update path ids
	peerID, idErr := getIDFromToken(peerTok)
	if idErr != nil {
		log.FromContext(ctx).Warnf("an error during parsing peer token: %+v", tokenErr)
	}
	id, idErr := getIDFromToken(tok)
	if idErr != nil {
		return nil, idErr
	}
	nse.PathIds = updatePathIds(nse.GetPathIds(), int(path.Index-1), peerID.String())
	nse.PathIds = updatePathIds(nse.GetPathIds(), int(path.Index), id.String())

	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
	path.Index = index
	return resp, err
}
