// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

// Package updatepath provides a chain element that sets the id of an incoming or outgoing request
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

type updatePathNSServer struct {
	tokenGenerator token.GeneratorFunc
}

// NewNetworkServiceRegistryServer - creates a new updatePath server to update NetworkService path.
func NewNetworkServiceRegistryServer(tokenGenerator token.GeneratorFunc) registry.NetworkServiceRegistryServer {
	return &updatePathNSServer{
		tokenGenerator: tokenGenerator,
	}
}

func (s *updatePathNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
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
	ns.PathIds = updatePathIds(ns.PathIds, int(path.Index-1), peerID.String())
	ns.PathIds = updatePathIds(ns.PathIds, int(path.Index), id.String())

	ns, err = next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
	if err != nil {
		return nil, err
	}
	path.Index = index

	return ns, nil
}

func (s *updatePathNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	ctx := server.Context()
	path := grpcmetadata.PathFromContext(ctx)

	// Update path
	peerTok, _, tokenErr := token.FromContext(ctx)
	if tokenErr != nil {
		log.FromContext(ctx).Warnf("an error during getting peer token from the context: %+v", tokenErr)
	}
	tok, _, tokenErr := generateToken(ctx, s.tokenGenerator)
	if tokenErr != nil {
		return errors.Wrap(tokenErr, "an error during generating token")
	}
	path, index, err := updatePath(path, peerTok, tok)
	if err != nil {
		return err
	}

	// Update path ids
	peerID, idErr := getIDFromToken(peerTok)
	if idErr != nil {
		log.FromContext(ctx).Warnf("an error during parsing peer token: %+v", tokenErr)
	}
	id, idErr := getIDFromToken(tok)
	if idErr != nil {
		return idErr
	}
	ns := query.NetworkService
	ns.PathIds = updatePathIds(ns.PathIds, int(path.Index-1), peerID.String())
	ns.PathIds = updatePathIds(ns.PathIds, int(path.Index), id.String())

	err = next.NetworkServiceRegistryServer(ctx).Find(query, server)
	if err != nil {
		return err
	}
	path.Index = index
	return nil
}

func (s *updatePathNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	path := grpcmetadata.PathFromContext(ctx)

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
	ns.PathIds = updatePathIds(ns.PathIds, int(path.Index-1), peerID.String())
	ns.PathIds = updatePathIds(ns.PathIds, int(path.Index), id.String())

	resp, err := next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
	path.Index = index
	return resp, err
}
