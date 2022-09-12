// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
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

// Package updatepath provides a chain element that sets the id of an incoming or outgoing request
package updatepath

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type updatePathServer struct {
	name string
}

// NewServer - creates a new updatePath client to update connection path.
//
//	name - name of the client
//
// Workflow are documented in common.go
func NewServer(name string) networkservice.NetworkServiceServer {
	return &updatePathServer{name: name}
}

func (i *updatePathServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (conn *networkservice.Connection, err error) {
	if request.Connection == nil {
		request.Connection = &networkservice.Connection{}
	}

	var index uint32
	request.Connection, index, err = updatePath(request.Connection, i.name)
	if err != nil {
		return nil, err
	}

	conn, err = next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	conn.Id = conn.Path.PathSegments[index].Id
	conn.Path.Index = index

	return conn, err
}

func (i *updatePathServer) Close(ctx context.Context, conn *networkservice.Connection) (_ *empty.Empty, err error) {
	conn, _, err = updatePath(conn, i.name)
	if err != nil {
		return nil, err
	}
	return next.Server(ctx).Close(ctx, conn)
}
