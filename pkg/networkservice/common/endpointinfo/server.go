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

// Package endpointinfo provides a chain element that adds pod, node and cluster names to request
package endpointinfo

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clientinfo"
)

type endpointInfo struct{}

// NewServer - creates a new networkservice.NetworkServiceServer chain element that adds pod, node and cluster names
// to request from corresponding environment variables
func NewServer() networkservice.NetworkServiceServer {
	return &endpointInfo{}
}

func (a *endpointInfo) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn := request.GetRequestConnection()
	if conn.Labels == nil {
		conn.Labels = make(map[string]string)
	}
	clientinfo.AddClientInfo(ctx, conn.Labels)
	return next.Client(ctx).Request(ctx, request)
}

func (a *endpointInfo) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn)
}
