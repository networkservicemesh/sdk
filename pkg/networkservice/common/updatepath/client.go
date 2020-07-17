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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type idClient struct {
	name string
}

// NewClient - creates a new setId client.
//             name - name of the client
//             Iff the current pathSegment name != name && pathSegment.id != networkservice.Id, set a new uuid for
//             connection id
func NewClient(name string) networkservice.NetworkServiceClient {
	return &idClient{name: name}
}

func (i *idClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (conn *networkservice.Connection, err error) {
	request.Connection, err = updatePath(request.Connection, i.name)
	if err != nil {
		return nil, err
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (i *idClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (_ *empty.Empty, err error) {
	conn, err = updatePath(conn, i.name)
	if err != nil {
		return nil, err
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
