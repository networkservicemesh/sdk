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

package updatetoken

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type updateTokenClient struct {
	tokenGenerator token.GeneratorFunc
}

// NewClient - creates a NetworkServiceClient chain element to update the Connection token information
func NewClient(tokenGenerator token.GeneratorFunc) networkservice.NetworkServiceClient {
	return &updateTokenClient{
		tokenGenerator: tokenGenerator,
	}
}

func (u *updateTokenClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if request.Connection == nil {
		request.Connection = &networkservice.Connection{}
	}
	err := updateToken(ctx, request.GetConnection(), u.tokenGenerator)
	index := request.GetConnection().GetPath().GetIndex()
	id := request.GetConnection().GetId()
	if err != nil {
		return nil, err
	}
	rv, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	rv.GetPath().Index = index
	rv.Id = id
	return rv, err
}

func (u *updateTokenClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	err := updateToken(ctx, conn, u.tokenGenerator)
	if err != nil {
		return nil, err
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
