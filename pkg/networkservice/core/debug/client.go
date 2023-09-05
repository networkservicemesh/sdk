// Copyright (c) 2023 Doc.ai and/or its affiliates.
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

package debug

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"
)

type beginDebugClient struct {
	debugged networkservice.NetworkServiceClient
}

type endDebugClient struct{}

const (
	clientRequestLoggedKey contextKeyType = "clientRequestLoggedKey"
	clientCloseLoggedKey   contextKeyType = "clientCloseLoggedKey"
	lastClientErrorKey     contextKeyType = "lastClientErrorKey"
	clientPrefix                          = "client"
)

// NewNetworkServiceClient - wraps tracing around the supplied networkservice.NetworkServiceClient
func NewNetworkServiceClient(debugged networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	return next.NewNetworkServiceClient(
		&beginDebugClient{debugged: debugged},
		&endDebugClient{},
	)
}

func (t *beginDebugClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// Create a new logger
	operation := typeutils.GetFuncName(t.debugged, methodNameRequest)
	updatedContext := withLog(ctx, request.GetConnection().GetId())

	if updatedContext.Value(clientRequestLoggedKey) == nil && isReadyForLogging(ctx, true) {
		updatedContext = context.WithValue(updatedContext, clientRequestLoggedKey, true)
		logRequest(updatedContext, request, clientPrefix, requestPrefix)
	}

	// Actually call the next
	rv, err := t.debugged.Request(updatedContext, request, opts...)
	if err != nil {
		lastError := updatedContext.Value(lastClientErrorKey)
		if lastError == nil || err.Error() != lastError {
			updatedContext = context.WithValue(updatedContext, lastServerErrorKey, err.Error())
			return nil, logError(updatedContext, err, operation)
		}
	}

	return rv, err
}

func (t *beginDebugClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	// Create a new logger
	operation := typeutils.GetFuncName(t.debugged, methodNameClose)
	updatedContext := withLog(ctx, conn.GetId())

	if updatedContext.Value(clientCloseLoggedKey) == nil && isReadyForLogging(ctx, true) {
		updatedContext = context.WithValue(updatedContext, clientCloseLoggedKey, true)
		logRequest(updatedContext, conn, clientPrefix, closePrefix)
	}

	// Actually call the next
	rv, err := t.debugged.Close(updatedContext, conn, opts...)
	if err != nil {
		lastError := updatedContext.Value(lastClientErrorKey)
		if lastError == nil || err.Error() != lastError {
			updatedContext = context.WithValue(updatedContext, lastServerErrorKey, err.Error())
			return nil, logError(updatedContext, err, operation)
		}
	}

	return rv, err
}

func (t *endDebugClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if isReadyForLogging(ctx, true) {
		nextClient := next.Client(ctx)
		withClientRequestTail(ctx, &nextClient)

		conn, err := nextClient.Request(ctx, request, opts...)

		tail, ok := clientRequestTail(ctx)
		if ok && &nextClient == tail {
			logResponse(ctx, conn, clientPrefix, requestPrefix)
		}

		return conn, err
	} else {
		return next.Client(ctx).Request(ctx, request, opts...)
	}
}

func (t *endDebugClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if isReadyForLogging(ctx, true) {
		nextClient := next.Client(ctx)
		withClientCloseTail(ctx, &nextClient)

		r, err := nextClient.Close(ctx, conn, opts...)

		tail, ok := clientCloseTail(ctx)
		if ok && &nextClient == tail {
			logResponse(ctx, conn, clientPrefix, closePrefix)
		}

		return r, err
	} else {
		return next.Client(ctx).Close(ctx, conn, opts...)
	}
}
