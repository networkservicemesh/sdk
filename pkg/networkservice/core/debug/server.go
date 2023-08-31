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

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"
)

type beginDebugServer struct {
	debugged networkservice.NetworkServiceServer
}

type endDebugServer struct{}

const (
	serverRequestLoggedKey contextKeyType = "serverRequestLoggedKey"
	serverCloseLoggedKey   contextKeyType = "serverCloseLoggedKey"
)

// NewNetworkServiceServer - wraps tracing around the supplied traced
func NewNetworkServiceServer(debugged networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return next.NewNetworkServiceServer(
		&beginDebugServer{debugged: debugged},
		&endDebugServer{},
	)
}

func (t *beginDebugServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// Create a new logger
	operation := typeutils.GetFuncName(t.debugged, methodNameRequest)
	updatedContext := withLog(ctx, operation, request.GetConnection().GetId())

	if updatedContext.Value(serverRequestLoggedKey) == nil {
		updatedContext = context.WithValue(updatedContext, serverRequestLoggedKey, true)
		logRequest(updatedContext, request, "server-request")
	}

	// Actually call the next
	rv, err := t.debugged.Request(updatedContext, request)
	if err != nil {
		return nil, logError(updatedContext, err, operation)
	}
	// logResponse(ctx, rv, "request")
	return rv, err
}

func (t *beginDebugServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// Create a new logger
	operation := typeutils.GetFuncName(t.debugged, methodNameClose)
	updatedContext := withLog(ctx, operation, conn.GetId())

	if updatedContext.Value(serverCloseLoggedKey) == nil {
		updatedContext = context.WithValue(updatedContext, serverCloseLoggedKey, true)
		logRequest(updatedContext, conn, "server-close")
	}

	// Actually call the next
	rv, err := t.debugged.Close(updatedContext, conn)
	if err != nil {
		return nil, logError(updatedContext, err, operation)
	}
	// logResponse(ctx, conn, "close")
	return rv, err
}

func (t *endDebugServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// logRequest(ctx, request, "request")

	nextServer := next.Server(ctx)
	withServerRequestTail(ctx, &nextServer)

	conn, err := nextServer.Request(ctx, request)

	tail, ok := serverRequestTail(ctx)
	if ok && &nextServer == tail {
		logResponse(ctx, conn, "server-request")
	}

	return conn, err
}

func (t *endDebugServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// logRequest(ctx, conn, "close")

	nextServer := next.Server(ctx)
	withServerCloseTail(ctx, &nextServer)

	r, err := nextServer.Close(ctx, conn)

	tail, ok := serverCloseTail(ctx)
	if ok && &nextServer == tail {
		logResponse(ctx, conn, "server-close")
	}

	return r, err
}
