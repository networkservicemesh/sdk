// Copyright (c) 2022 Doc.ai and/or its affiliates.
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

package policyroute

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// PolicyRoutesFunc - method for the new policyRoutes getting.
type PolicyRoutesFunc func() []*networkservice.PolicyRoute

type policyrouteServer struct {
	getPolicies PolicyRoutesFunc
}

// NewServer creates a NetworkServiceServer that will put the routing policies to connection context.
func NewServer(policyRouteGetter PolicyRoutesFunc) networkservice.NetworkServiceServer {
	return &policyrouteServer{
		getPolicies: policyRouteGetter,
	}
}

func (p *policyrouteServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn := request.GetConnection()
	if conn.GetContext() == nil {
		conn.Context = &networkservice.ConnectionContext{}
	}
	if conn.GetContext().GetIpContext() == nil {
		conn.GetContext().IpContext = &networkservice.IPContext{}
	}
	ipContext := conn.GetContext().GetIpContext()

	// Update policies
	// Remove old IP addresses
	policies := p.getPolicies()
	for _, p := range ipContext.GetPolicies() {
		if p.GetFrom() != "" {
			for s := range ipContext.GetSrcIpAddrs() {
				if ipContext.GetSrcIpAddrs()[s] == p.GetFrom() {
					ipContext.SrcIpAddrs = append(ipContext.SrcIpAddrs[:s], ipContext.GetSrcIpAddrs()[s+1:]...)
					break
				}
			}
		}
	}
	// Use new policies
	ipContext.Policies = policies

	// Add new IP addresses
	for _, p := range policies {
		if p.GetFrom() != "" {
			ipContext.SrcIpAddrs = append(ipContext.SrcIpAddrs, p.GetFrom())
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (p *policyrouteServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
