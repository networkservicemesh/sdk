// Copyright (c) 2020 Cisco Systems, Inc.
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

	"github.com/networkservicemesh/sdk/pkg/tools/opa"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type authorizeServer struct {
	policies []opa.AuthorizationPolicy
}

// NewServer - returns a new authorization networkservicemesh.NetworkServiceServers
func NewServer(opts ...ServerOption) networkservice.NetworkServiceServer {
	s := &authorizeServer{}
	for _, o := range opts {
		o.apply(s)
	}
	return s
}

func (a *authorizeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if err := a.check(ctx, request.GetConnection()); err != nil {
		return nil, err
	}
	return next.Server(ctx).Request(ctx, request)
}

func (a *authorizeServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if err := a.check(ctx, conn); err != nil {
		return nil, err
	}
	return next.Server(ctx).Close(ctx, conn)
}

func (a *authorizeServer) check(ctx context.Context, conn *networkservice.Connection) error {
	for _, p := range a.policies {
		if err := p.Check(ctx, conn.GetPath()); err != nil {
			return err
		}
	}
	return nil
}
