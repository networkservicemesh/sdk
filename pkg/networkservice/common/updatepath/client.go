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

package updatepath

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type updatePathClient struct {
	commonUpdatePath
}

// NewClient - creates a NetworkServiceClient chain element to update the Connection.Path
//             - name - the name of the NetworkServiceClient of which the chain element is part
func NewClient(name string, tokenGenerator token.GeneratorFunc) networkservice.NetworkServiceClient {
	return &updatePathClient{
		commonUpdatePath: commonUpdatePath{
			name:           name,
			tokenGenerator: tokenGenerator,
		},
	}
}

func (u *updatePathClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	err := u.updatePath(ctx, request.GetConnection())
	index := request.GetConnection().GetPath().GetIndex()
	if err != nil {
		return nil, err
	}
	rv, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	rv.GetPath().Index = index
	return rv, err
}

func (u *updatePathClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	err := u.updatePath(ctx, conn)
	if err != nil {
		return nil, err
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
