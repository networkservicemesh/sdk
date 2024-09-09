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

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"
)

type beginConciseServer struct {
	traced networkservice.NetworkServiceServer
}

type endConciseServer struct{}

const (
	serverPrefix = "server"
)

// NewNetworkServiceServer - wraps tracing around the supplied traced.
func NewNetworkServiceServer(traced networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return next.NewNetworkServiceServer(
		&beginConciseServer{traced: traced},
		&endConciseServer{},
	)
}

func (t *beginConciseServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// Create a new logger
	connID := request.GetConnection().GetId()
	if request.GetConnection().GetPath() != nil && len(request.GetConnection().GetPath().GetPathSegments()) > 0 {
		connID = request.GetConnection().GetPath().GetPathSegments()[0].GetId()
	}

	var tail networkservice.NetworkServiceServer
	if ctx, tail = serverRequestTail(ctx); tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, connID, methodNameRequest)
		defer finish()
		logRequest(ctx, request, serverPrefix, requestPrefix)
	}

	// Actually call the next
	rv, err := t.traced.Request(ctx, request)
	if err != nil {
		lastError := loadAndStoreServerRequestError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameRequest)
			return nil, logError(ctx, err, operation)
		}
	}

	return rv, err
}

func (t *beginConciseServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// Create a new logger
	connID := conn.GetId()
	if conn.GetPath() != nil && len(conn.GetPath().GetPathSegments()) > 0 {
		connID = conn.GetPath().GetPathSegments()[0].GetId()
	}

	var tail networkservice.NetworkServiceServer
	if ctx, tail = serverCloseTail(ctx); tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, connID, methodNameClose)
		defer finish()

		logRequest(ctx, conn, serverPrefix, closePrefix)
	}

	// Actually call the next
	rv, err := t.traced.Close(ctx, conn)
	if err != nil {
		lastError := loadAndStoreServerCloseError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameClose)
			return nil, logError(ctx, err, operation)
		}
	}

	return rv, err
}

func (t *endConciseServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	nextServer := next.Server(ctx)
	ctx = withServerRequestTail(ctx, nextServer)

	conn, err := nextServer.Request(ctx, request)

	var tail networkservice.NetworkServiceServer
	if ctx, tail = serverRequestTail(ctx); tail == nextServer {
		logResponse(ctx, conn, serverPrefix, requestPrefix)
	}
	return conn, err
}

func (t *endConciseServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	nextServer := next.Server(ctx)
	ctx = withServerCloseTail(ctx, nextServer)

	r, err := nextServer.Close(ctx, conn)

	var tail networkservice.NetworkServiceServer
	if ctx, tail = serverCloseTail(ctx); tail == nextServer {
		logResponse(ctx, conn, serverPrefix, closePrefix)
	}
	return r, err
}
