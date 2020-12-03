// Copyright (c) 2020 Cisco and/or its affiliates.
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

package metadata

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type metaDataClient struct {
	Map metaDataMap
}

// NewClient - Enable per Connection.Id metadata for the client
//             Must come after updatepath.NewClient() in the chain
func NewClient() networkservice.NetworkServiceClient {
	return &metaDataClient{}
}

func (m *metaDataClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	connID := request.GetConnection().GetId()

	ctx = store(ctx, connID, &m.Map)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		del(ctx, connID, &m.Map)
		return nil, err
	}

	return conn, nil
}

func (m *metaDataClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	ctx = del(ctx, conn.GetId(), &m.Map)
	return next.Client(ctx).Close(ctx, conn, opts...)
}
