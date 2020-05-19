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
	"github.com/networkservicemesh/sdk/pkg/tools/opautils"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/open-policy-agent/opa/rego"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type authorizeServer struct {
	p *rego.PreparedEvalQuery
}

// NewServer - returns a new authorization networkservicemesh.NetworkServiceServers
func NewServer(p *rego.PreparedEvalQuery) networkservice.NetworkServiceServer {
	return &authorizeServer{
		p: p,
	}
}

func (a *authorizeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	var authInfo credentials.AuthInfo
	if p, ok := peer.FromContext(ctx); ok {
		authInfo = p.AuthInfo
	}
	input, err := opautils.PreparedOpaInput(request.Connection, authInfo, opautils.Request, opautils.Endpoint)
	if err != nil {
		return nil, err
	}
	if err := opautils.CheckPolicy(ctx, a.p, input); err != nil {
		return nil, err
	}
	return next.Server(ctx).Request(ctx, request)
}

func (a *authorizeServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	var authInfo credentials.AuthInfo
	if p, ok := peer.FromContext(ctx); ok {
		authInfo = p.AuthInfo
	}
	input, err := opautils.PreparedOpaInput(conn, authInfo, opautils.Close, opautils.Endpoint)
	if err != nil {
		return nil, err
	}
	if err := opautils.CheckPolicy(ctx, a.p, input); err != nil {
		return nil, err
	}
	return next.Server(ctx).Close(ctx, conn)
}
