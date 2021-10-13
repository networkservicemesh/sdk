// Copyright (c) 2021 Cisco and/or its affiliates.
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

// Package clientconn - chain element for injecting a grpc.ClientConnInterface into the client chain
package clientconn

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// NewClient - returns a clientconn chain element
func NewClient(cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
	return &clientConnClient{
		cc: cc,
	}
}

type clientConnClient struct {
	cc grpc.ClientConnInterface
}

func (c *clientConnClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if c.cc != nil {
		_, _ = LoadOrStore(ctx, c.cc)
	}
	return next.Client(ctx).Request(ctx, request)
}

func (c *clientConnClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	_, err := next.Client(ctx).Close(ctx, conn)
	if c.cc != nil {
		cc, loaded := LoadAndDelete(ctx)
		if loaded && cc != c.cc {
			Store(ctx, cc)
		}
	}
	return &emptypb.Empty{}, err
}
