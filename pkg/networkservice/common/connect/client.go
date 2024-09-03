// Copyright (c) 2021 Cisco and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

package connect

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
)

type connectClient struct{}

// NewClient - returns a connect chain element.
func NewClient() networkservice.NetworkServiceClient {
	return &connectClient{}
}

func (c *connectClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	cc, loaded := clientconn.Load(ctx)
	if !loaded {
		return nil, errors.New("no grpc.ClientConnInterface provided")
	}
	return networkservice.NewNetworkServiceClient(cc).Request(ctx, request, opts...)
}

func (c *connectClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cc, loaded := clientconn.Load(ctx)
	if !loaded {
		return nil, errors.New("no grpc.ClientConnInterface provided")
	}
	return networkservice.NewNetworkServiceClient(cc).Close(ctx, conn, opts...)
}
