// Copyright (c) 2023 Cisco and/or its affiliates.
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

package traceconcise

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"
)

type beginConciseClient struct {
	traced networkservice.NetworkServiceClient
}

type endConciseClient struct{}

const (
	clientPrefix = "client"
)

// NewNetworkServiceClient - wraps tracing around the supplied networkservice.NetworkServiceClient.
func NewNetworkServiceClient(traced networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	return next.NewNetworkServiceClient(
		&beginConciseClient{traced: traced},
		&endConciseClient{},
	)
}

func (t *beginConciseClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// Create a new logger
	connID := request.GetConnection().GetId()
	if request.GetConnection().GetPath() != nil && len(request.GetConnection().GetPath().GetPathSegments()) > 0 {
		connID = request.GetConnection().GetPath().GetPathSegments()[0].GetId()
	}

	var tail networkservice.NetworkServiceClient
	if ctx, tail = clientRequestTail(ctx); tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, connID, methodNameRequest)
		defer finish()
		logRequest(ctx, request, clientPrefix, requestPrefix)
	}

	// Actually call the next
	rv, err := t.traced.Request(ctx, request, opts...)
	if err != nil {
		lastError := loadAndStoreClientRequestError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameRequest)
			return nil, logError(ctx, err, operation)
		}
	}

	return rv, err
}

func (t *beginConciseClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	// Create a new logger
	connID := conn.GetId()
	if conn.GetPath() != nil && len(conn.GetPath().GetPathSegments()) > 0 {
		connID = conn.GetPath().GetPathSegments()[0].GetId()
	}

	var tail networkservice.NetworkServiceClient
	if ctx, tail = clientCloseTail(ctx); tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, connID, methodNameClose)
		defer finish()
		logRequest(ctx, conn, clientPrefix, closePrefix)
	}

	// Actually call the next
	rv, err := t.traced.Close(ctx, conn, opts...)
	if err != nil {
		lastError := loadAndStoreClientCloseError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameClose)
			return nil, logError(ctx, err, operation)
		}
	}

	return rv, err
}

func (t *endConciseClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	nextClient := next.Client(ctx)
	ctx = withClientRequestTail(ctx, nextClient)

	conn, err := nextClient.Request(ctx, request, opts...)

	var tail networkservice.NetworkServiceClient
	if ctx, tail = clientRequestTail(ctx); tail == nextClient {
		logResponse(ctx, conn, clientPrefix, requestPrefix)
	}
	return conn, err
}

func (t *endConciseClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	nextClient := next.Client(ctx)
	ctx = withClientCloseTail(ctx, nextClient)

	r, err := nextClient.Close(ctx, conn, opts...)

	var tail networkservice.NetworkServiceClient
	if ctx, tail = clientCloseTail(ctx); tail == nextClient {
		logResponse(ctx, conn, clientPrefix, closePrefix)
	}
	return r, err
}
