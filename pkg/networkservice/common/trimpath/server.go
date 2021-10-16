// Copyright (c) 2021 Cisco and/or its affiliates.
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

package trimpath

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type trimpathServer struct{}

// NewServer - returns a trimpath server chain element that will truncate the Connection.Path at the server
// that contains it *unless* it is part of a passthrough that utilizes a client containing trimpath.NewClient()
func NewServer() networkservice.NetworkServiceServer {
	return &trimpathServer{}
}

func (t *trimpathServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	if shouldTrimPath(ctx) {
		conn.GetPath().PathSegments = conn.GetPath().GetPathSegments()[:conn.GetPath().GetIndex()+1]
	}
	return conn, nil
}

func (t *trimpathServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
