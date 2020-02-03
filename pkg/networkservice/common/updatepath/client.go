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
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type updatePathClient struct {
	name string
}

// NewClient - creates a NetworkServiceClient chain element to update the Connection.Path
//             - name - the name of the NetworkServiceClient of which the chain element is part
func NewClient(name string) networkservice.NetworkServiceClient {
	return &updatePathClient{name: name}
}

func (u *updatePathClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// Handle zero index case
	if int(request.GetConnection().GetPath().GetIndex()) == len(request.GetConnection().GetPath().GetPathSegments()) {
		request.GetConnection().GetPath().PathSegments = append(request.GetConnection().GetPath().PathSegments, &networkservice.PathSegment{})
	}
	if int(request.GetConnection().GetPath().GetIndex()) >= len(request.GetConnection().GetPath().GetPathSegments()) {
		return nil, errors.Errorf("NetworkServiceRequest.Connection.Path.Index(%d) >= len(NetworkServiceRequest.Connection.Path.PathSegments)(%d)",
			request.GetConnection().GetPath().GetIndex(),
			len(request.GetConnection().GetPath().GetPathSegments()))
	}
	request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].Name = u.name
	request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].Id = request.GetConnection().GetId()
	// TODO set token and expiration
	// request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].Token =
	// request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].Expires =
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (u *updatePathClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
