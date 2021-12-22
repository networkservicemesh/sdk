// Copyright (c) 2020-2021 Cisco Systems, Inc.
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

// Package clienturl provides server chain element that sets client URL in context
package opentelemetry

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type metricsServer struct {
}

// NewServer - returns a new server chain element that sets client URL in context
func NewServer() networkservice.NetworkServiceServer {
	return &metricsServer{}
}

func (c *metricsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	//request.Connection.GetCurrentPathSegment().Metrics = make(map[string]string)
	//request.Connection.GetCurrentPathSegment().Metrics["my_nsm_metric"] = "10000"
	return next.Server(ctx).Request(ctx, request)
}

func (c *metricsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
