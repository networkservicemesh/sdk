// Copyright (c) 2020-2022 Cisco Systems, Inc.
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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type updatePathClient struct {
	name string
}

// NewClient - creates a new updatePath client to update connection path.
//
//	name - name of the client
//
// Workflow are documented in common.go
func NewClient(name string) networkservice.NetworkServiceClient {
	return &updatePathClient{name: name}
}

func (i *updatePathClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (conn *networkservice.Connection, err error) {
	if request.Connection == nil {
		request.Connection = &networkservice.Connection{}
	}

	var index uint32
	request.Connection, index, err = updatePath(request.Connection, i.name)
	if err != nil {
		return nil, err
	}

	conn, err = next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	conn.Id = conn.Path.PathSegments[index].Id
	conn.Path.Index = index

	return conn, err
}

func (i *updatePathClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (_ *empty.Empty, err error) {
	conn, _, err = updatePath(conn, i.name)
	if err != nil {
		return nil, err
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
