// Copyright (c) 2023-2024 Cisco and/or its affiliates.
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

// Package trace provides wrappers for tracing around a networkservice.NetworkServiceClient
package trace

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace/traceconcise"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace/traceverbose"
)

type traceClient struct {
	verbose  networkservice.NetworkServiceClient
	concise  networkservice.NetworkServiceClient
	original networkservice.NetworkServiceClient
}

// NewNetworkServiceClient - wraps tracing around the supplied networkservice.NetworkServiceClient
func NewNetworkServiceClient(traced networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	return &traceClient{
		verbose:  traceverbose.NewNetworkServiceClient(traced),
		concise:  traceconcise.NewNetworkServiceClient(traced),
		original: traced,
	}
}

func (t *traceClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if logrus.GetLevel() <= logrus.WarnLevel {
		return t.original.Request(ctx, request)
	}
	if logrus.GetLevel() >= logrus.DebugLevel {
		return t.verbose.Request(ctx, request, opts...)
	}
	return t.concise.Request(ctx, request, opts...)
}

func (t *traceClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if logrus.GetLevel() <= logrus.WarnLevel {
		return t.original.Close(ctx, conn)
	}
	if logrus.GetLevel() >= logrus.DebugLevel {
		return t.verbose.Close(ctx, conn, opts...)
	}
	return t.concise.Close(ctx, conn, opts...)
}
