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

// Package updatepath provides chain elements to update Connection.Path
package updatepath

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type updatePathServer struct {
	name string
}

// NewServer - creates a NetworkServiceServer chain element to update the Connection.Path
//             - name - the name of the NetworkServiceServer of which the chain element is part
func NewServer(name string) networkservice.NetworkServiceServer {
	return &updatePathServer{name: name}
}

func (u *updatePathServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	path := request.GetConnection().GetPath()
	if int(path.GetIndex()) >= len(path.GetPathSegments()) {
		return nil, errors.Errorf("NetworkServiceRequest.Connection.Path.Index(%d) >= len(NetworkServiceRequest.Connection.Path.PathSegments)(%d)",
			path.GetIndex(), len(path.GetPathSegments()))
	}
	// increment the index
	path.Index++
	// extend the path (presuming that we need to)
	if int(path.GetIndex()) == len(path.GetPathSegments()) {
		path.PathSegments = append(path.PathSegments, &networkservice.PathSegment{})
	}
	path.GetPathSegments()[path.GetIndex()].Name = u.name
	path.GetPathSegments()[path.GetIndex()].Id = request.GetConnection().GetId()
	// TODO set token and expiration
	// request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].Token =
	// request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].Expires =
	return next.Server(ctx).Request(ctx, request)
}

func (u *updatePathServer) Close(context.Context, *networkservice.Connection) (*empty.Empty, error) {
	panic("implement me")
}
