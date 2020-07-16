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

// Package adapters provides adapters to translate between networkservice.NetworkService{Server,Client}
package adapters

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type serverToClient struct {
	server networkservice.NetworkServiceServer
}

// NewServerToClient - returns a new networkservice.NetworkServiceClient that is a wrapper around server
func NewServerToClient(server networkservice.NetworkServiceServer) networkservice.NetworkServiceClient {
	return &serverToClient{server: next.NewNetworkServiceServer(server, &contextServer{})}
}

func (s *serverToClient) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, _ ...grpc.CallOption) (*networkservice.Connection, error) {
	doneCtx := withCapturedContext(ctx)
	conn, err := s.server.Request(doneCtx, in)
	if err != nil {
		return nil, err
	}
	lastCtx := getCapturedContext(doneCtx)
	if lastCtx == nil {
		return conn, nil
	}
	if in == nil {
		in = &networkservice.NetworkServiceRequest{}
	}
	in.Connection = conn
	return next.Client(ctx).Request(lastCtx, in)
}

func (s *serverToClient) Close(ctx context.Context, in *networkservice.Connection, _ ...grpc.CallOption) (*empty.Empty, error) {
	doneCtx := withCapturedContext(ctx)
	conn, err := s.server.Close(doneCtx, in)
	if err != nil {
		return nil, err
	}
	lastCtx := getCapturedContext(doneCtx)
	if lastCtx == nil {
		return conn, nil
	}
	return next.Client(ctx).Close(lastCtx, in)
}
