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

package setid

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/connection"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type idClient struct {
	name string
}

// NewClient - creates a new setId client.
//             name - name of the client
//             Iff the current pathSegment name != name && pathSegment.id != connection.Id, set a new uuid for
//             connection id
func NewClient(name string) networkservice.NetworkServiceClient {
	return &idClient{name: name}
}

func (i *idClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	pathSegments := request.GetConnection().GetPath().GetPathSegments()
	index := request.GetConnection().GetPath().GetIndex()
	if len(pathSegments) > int(index) &&
		pathSegments[index].GetName() != i.name &&
		pathSegments[index].GetId() != request.GetConnection().GetId() {
		request.GetConnection().Id = uuid.New().String()
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (i *idClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
