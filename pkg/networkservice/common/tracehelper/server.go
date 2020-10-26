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

// Package tracehelper provides NetworkService{Client, Server} for injecting ConnectionInfo into context
// for trace chain-element needs
package tracehelper

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// traceHelperServer NetworkServiceServer, that injects ConnectionInfo into context
type traceHelperServer struct {
}

// NewServer - creates a new traceHelper server to inject ConnectionInfo into context.
func NewServer() networkservice.NetworkServiceServer {
	return &traceHelperServer{}
}

func (th *traceHelperServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	ctx = withConnectionInfo(ctx, &ConnectionInfo{})
	return next.Server(ctx).Request(ctx, request)
}

func (th *traceHelperServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	ctx = withConnectionInfo(ctx, &ConnectionInfo{})
	return next.Server(ctx).Close(ctx, conn)
}
