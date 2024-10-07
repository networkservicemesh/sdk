// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023-2024 Cisco and/or its affiliates.
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

// Package interdomainbypass injects into incoming context the URL to remote side only if requesting endpoint has been resolved.
package interdomainbypass

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

type interdomainBypassServer struct{}

// NewServer - returns a new NetworkServiceServer that injects the URL to remote side into context on requesting resolved endpoint
func NewServer() networkservice.NetworkServiceServer {
	return new(interdomainBypassServer)
}

func (n *interdomainBypassServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	originalNSEName := request.GetConnection().NetworkServiceEndpointName
	originalNS := request.GetConnection().NetworkService
	request.GetConnection().NetworkServiceEndpointName = interdomain.Target(originalNSEName)
	request.GetConnection().NetworkService = interdomain.Target(originalNS)
	resp, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	resp.NetworkServiceEndpointName = originalNSEName
	resp.NetworkService = originalNS
	return resp, nil
}

func (n *interdomainBypassServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	originalNSEName := conn.GetNetworkServiceEndpointName()
	originalNS := conn.GetNetworkService()
	conn.NetworkServiceEndpointName = interdomain.Target(originalNSEName)
	conn.NetworkService = interdomain.Target(originalNS)
	resp, err := next.Server(ctx).Close(ctx, conn)
	if err != nil {
		return nil, err
	}
	conn.NetworkServiceEndpointName = originalNSEName
	conn.NetworkService = originalNS
	return resp, nil
}
