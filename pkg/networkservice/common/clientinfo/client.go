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

// Package clientinfo provides a chain element that adds pod, node and cluster names to request
package clientinfo

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/clientinfo"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type clientInfo struct{}

// NewClient - creates a new networkservice.NetworkServiceClient chain element that adds pod, node and cluster names
// to request from corresponding environment variables.
func NewClient() networkservice.NetworkServiceClient {
	return &clientInfo{}
}

func (a *clientInfo) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn := request.GetRequestConnection()
	if conn.Labels == nil {
		conn.Labels = make(map[string]string)
	}
	clientinfo.AddClientInfo(ctx, conn.GetLabels())
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (a *clientInfo) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
