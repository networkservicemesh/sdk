// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

// Package selectservice provides a chain element that can choose which element to call based on network service name
package selectservice

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type selectServiceServer struct {
	servers map[string]networkservice.NetworkServiceServer
}

// NewServer - returns new NetworkServiceServer chain element that will
// select one of the servers passed in arguments based on the requested network service name
//
//  servers - map for supported network services.
//
func NewServer(servers map[string]networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	rv := &selectServiceServer{
		servers: servers,
	}
	return rv
}

func (s *selectServiceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	srv, ok := s.servers[request.GetConnection().GetNetworkService()]
	if !ok || srv == nil {
		return nil, errors.New("no server found for specified service and no default is provided")
	}
	return srv.Request(ctx, request)
}

func (s *selectServiceServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	srv, ok := s.servers[conn.GetNetworkService()]
	if !ok || srv == nil {
		return nil, errors.New("no server found for specified service and no default is provided")
	}
	return srv.Close(ctx, conn)
}
