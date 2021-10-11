// Copyright (c) 2020 Cisco Systems, Inc.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package trace

import (
	"context"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"
)

type beginTraceClient struct {
	traced networkservice.NetworkServiceClient
}

type endTraceClient struct{}

// NewNetworkServiceClient - wraps tracing around the supplied networkservice.NetworkServiceClient
func NewNetworkServiceClient(traced networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	return next.NewNetworkServiceClient(
		&beginTraceClient{traced: traced},
		&endTraceClient{},
	)
}

func (t *beginTraceClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// Create a new logger
	fields := make(map[string]interface{})
	for k, v := range log.Fields(ctx) {
		fields[k] = v
	}

	// don't change type if it's already present - it happens when registry elements used in endpoint discovery
	if _, ok := fields["type"]; !ok {
		fields["type"] = "NetworkService"
		ctx = log.WithFields(ctx, fields)
	}

	if len(request.GetConnection().GetId()) > 0 {
		fields["id"] = request.GetConnection().GetId()
		ctx = log.WithFields(ctx, fields)
	}

	operation := typeutils.GetFuncName(t.traced, "Request")
	ctx, finish := withLog(ctx, operation)
	defer finish()

	logRequest(ctx, request, "request")
	// Actually call the next
	rv, err := t.traced.Request(ctx, request, opts...)
	if err != nil {
		return nil, logError(ctx, err, operation)
	}
	logResponse(ctx, rv, "request")
	return rv, err
}

func (t *beginTraceClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	// Create a new logger
	fields := make(map[string]interface{})
	for k, v := range log.Fields(ctx) {
		fields[k] = v
	}

	// don't change type if it's already present - it happens when registry elements used in endpoint discovery
	if _, ok := fields["type"]; !ok {
		fields["type"] = "NetworkService"
		ctx = log.WithFields(ctx, fields)
	}

	if len(conn.GetId()) > 0 {
		fields["id"] = conn.GetId()
		ctx = log.WithFields(ctx, fields)
	}

	operation := typeutils.GetFuncName(t.traced, "Close")
	ctx, finish := withLog(ctx, operation)
	defer finish()

	logRequest(ctx, conn, "close")
	rv, err := t.traced.Close(ctx, conn, opts...)
	if err != nil {
		return nil, logError(ctx, err, operation)
	}
	logResponse(ctx, conn, "close")

	return rv, err
}

func (t *endTraceClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	logRequest(ctx, request, "request")
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	logResponse(ctx, conn, "request")
	return conn, err
}

func (t *endTraceClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	logRequest(ctx, conn, "close")
	r, err := next.Client(ctx).Close(ctx, conn, opts...)
	logResponse(ctx, conn, "close")
	return r, err
}
