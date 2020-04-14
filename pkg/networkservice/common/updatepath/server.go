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
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type updatePathServer struct {
	commonUpdatePath
}

// NewServer - creates a NetworkServiceServer chain element to update the Connection.Path
//             - name - the name of the NetworkServiceServer of which the chain element is part
func NewServer(name string, tokenGenerator token.GeneratorFunc) networkservice.NetworkServiceServer {
	return &updatePathServer{
		commonUpdatePath{
			name:           name,
			tokenGenerator: tokenGenerator,
		},
	}
}

func (u *updatePathServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	err := u.updatePath(ctx, request.GetConnection())
	if err != nil {
		return nil, err
	}
	return next.Server(ctx).Request(ctx, request)
}

func (u *updatePathServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	err := u.updatePath(ctx, conn)
	if err != nil {
		return nil, err
	}
	return next.Server(ctx).Close(ctx, conn)
}
