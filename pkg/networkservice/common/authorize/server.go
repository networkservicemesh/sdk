// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2025 Cisco Systems, Inc.
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

// Package authorize provides authz checks for incoming or returning connections.
package authorize

import (
	"context"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
	"github.com/networkservicemesh/sdk/pkg/tools/spire"
)

type authorizeServer struct {
	policies              policiesList
	spiffeIDConnectionMap *genericsync.Map[spiffeid.ID, *genericsync.Map[string, struct{}]]
}

// NewServer - returns a new authorization networkservicemesh.NetworkServiceServers
// Authorize server checks left side of Path.
func NewServer(opts ...Option) networkservice.NetworkServiceServer {
	o := &options{
		policyPaths: []string{
			"etc/nsm/opa/common/.*.rego",
			"etc/nsm/opa/server/.*.rego",
		},
		spiffeIDConnectionMap: &genericsync.Map[spiffeid.ID, *genericsync.Map[string, struct{}]]{},
	}
	for _, opt := range opts {
		opt(o)
	}

	policies, err := opa.PoliciesByFileMask(o.policyPaths...)
	if err != nil {
		panic(errors.Wrap(err, "failed to read policies in NetworkService authorize client").Error())
	}
	var policyList policiesList
	for _, p := range policies {
		policyList = append(policyList, p)
	}

	var s = &authorizeServer{
		policies:              policyList,
		spiffeIDConnectionMap: o.spiffeIDConnectionMap,
	}
	return s
}

func (a *authorizeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn := request.GetConnection()
	var index = conn.GetPath().GetIndex()
	var leftSide = &networkservice.Path{
		Index:        index,
		PathSegments: conn.GetPath().GetPathSegments()[:index+1],
	}
	if _, ok := peer.FromContext(ctx); ok {
		if err := a.policies.check(ctx, leftSide); err != nil {
			return nil, err
		}
	}

	spiffeID, loadErr := spire.PeerSpiffeIDFromContext(ctx)
	if loadErr == nil {
		connID := conn.GetPath().GetPathSegments()[index-1].GetId()
		ids, ok := a.spiffeIDConnectionMap.Load(spiffeID)
		if !ok {
			ids = new(genericsync.Map[string, struct{}])
		}
		ids.Store(connID, struct{}{})
		a.spiffeIDConnectionMap.Store(spiffeID, ids)
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if loadErr == nil && err != nil {
		a.spiffeIDConnectionMap.Delete(spiffeID)
	}

	return conn, err
}

func (a *authorizeServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	var index = conn.GetPath().GetIndex()
	var leftSide = &networkservice.Path{
		Index:        index,
		PathSegments: conn.GetPath().GetPathSegments()[:index+1],
	}
	if spiffeID, err := spire.PeerSpiffeIDFromContext(ctx); err == nil {
		connID := conn.GetPath().GetPathSegments()[index-1].GetId()
		ids, ok := a.spiffeIDConnectionMap.Load(spiffeID)
		idsEmpty := true
		if ok {
			if _, loaded := ids.Load(connID); loaded {
				ids.Delete(connID)
			}

			ids.Range(func(_ string, _ struct{}) bool {
				idsEmpty = false
				return false
			})
		}

		if idsEmpty {
			a.spiffeIDConnectionMap.Delete(spiffeID)
		} else {
			a.spiffeIDConnectionMap.Store(spiffeID, ids)
		}
	}
	if _, ok := peer.FromContext(ctx); ok {
		if err := a.policies.check(ctx, leftSide); err != nil {
			return nil, err
		}
	}
	return next.Server(ctx).Close(ctx, conn)
}
