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

// Package setid provides a chain element that sets the id of an incoming or outgoing request
package setid

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservicemesh/core/next"
)

type idServer struct {
	name string
}

// NewServer - creates a new setId server.
//             name - name of the client
//             Iff the current pathSegment name != name && pathsegment.id != connection.Id, set a new uuid for
//             connection id
func NewServer(name string) networkservice.NetworkServiceServer {
	return &idServer{name: name}
}

func (i *idServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	pathSegments := request.GetConnection().GetPath().GetPathSegments()
	index := request.GetConnection().GetPath().GetIndex()
	if len(pathSegments) > int(index) &&
		pathSegments[index].GetName() != i.name &&
		pathSegments[index].GetId() != request.GetConnection().GetId() {
		request.GetConnection().Id = uuid.New().String()
	}
	return next.Server(ctx).Request(ctx, request)
}

func (i *idServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
