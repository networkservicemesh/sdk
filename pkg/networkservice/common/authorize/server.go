// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2021 Cisco Systems, Inc.
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

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

type authorizeServer struct {
	policies policiesList
}

// NewServer - returns a new authorization networkservicemesh.NetworkServiceServers
func NewServer(opts ...Option) networkservice.NetworkServiceServer {
	var s = &authorizeServer{
		policies: []Policy{
			opa.WithTokensValidPolicy(),
			opa.WithPrevTokenSignedPolicy(),
			opa.WithTokensExpiredPolicy(),
			opa.WithTokenChainPolicy(),
		},
	}
	for _, o := range opts {
		o.apply(&s.policies)
	}
	return s
}

func (a *authorizeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if err := a.policies.check(ctx, request.GetConnection()); err != nil {
		return nil, err
	}
	return next.Server(ctx).Request(ctx, request)
}

func (a *authorizeServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if err := a.policies.check(ctx, conn); err != nil {
		return nil, err
	}
	return next.Server(ctx).Close(ctx, conn)
}
