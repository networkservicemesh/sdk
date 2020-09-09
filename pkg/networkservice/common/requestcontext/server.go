// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package requestcontext provides a NetworkService* to fill all required client request fields.
package requestcontext

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type requestServer struct {
}

// NewServer - creates a Server that will fill all required connection request fields
func NewServer() networkservice.NetworkServiceServer {
	return &requestServer{}
}

func (u *requestServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.Connection == nil {
		request.Connection = &networkservice.Connection{}
	}
	if request.Connection.Context == nil {
		request.Connection.Context = &networkservice.ConnectionContext{}
	}

	return next.Server(ctx).Request(ctx, request)
}

func (u *requestServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
