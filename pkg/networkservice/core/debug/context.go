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

// Package debug provides a wrapper for logging around a networkservice.NetworkServiceClient
package debug

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
)

type contextKeyType string

const (
	serverChainRequestTailKey contextKeyType = "serverChainRequestTailKey"
	serverChainCloseTailKey   contextKeyType = "serverChainCloseTailKey"
	serverRequestErrorKey     contextKeyType = "serverRequestErrorKey"
	serverCloseErrorKey       contextKeyType = "serverCloseErrorKey"
	clientRequestErrorKey     contextKeyType = "clientRequestErrorKey"
	clientCloseErrorKey       contextKeyType = "clientCloseErrorKey"
	clientChainRequestTailKey contextKeyType = "clientChainRequestTailKey"
	clientChainCloseTailKey   contextKeyType = "clientChainCloseTailKey"
	loggedType                string         = "networkService"
)

func withLog(parent context.Context, connectionID string) (c context.Context) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}

	fields := []*log.Field{log.NewField("id", connectionID), log.NewField("type", loggedType)}
	lLogger := logruslogger.LoggerWithFields(fields)

	return log.WithLog(parent, lLogger)
}

func withServerRequestTail(ctx context.Context, server *networkservice.NetworkServiceServer) {
	metadata.Map(ctx, false).Store(serverChainRequestTailKey, server)
}

func serverRequestTail(ctx context.Context) (server *networkservice.NetworkServiceServer, ok bool) {
	rawValue, ok := metadata.Map(ctx, false).Load(serverChainRequestTailKey)
	if !ok {
		return
	}
	value, ok := rawValue.(*networkservice.NetworkServiceServer)
	return value, ok
}

func withServerCloseTail(ctx context.Context, server *networkservice.NetworkServiceServer) {
	metadata.Map(ctx, false).Store(serverChainCloseTailKey, server)
}

func serverCloseTail(ctx context.Context) (server *networkservice.NetworkServiceServer, ok bool) {
	rawValue, ok := metadata.Map(ctx, false).Load(serverChainCloseTailKey)
	if !ok {
		return
	}
	value, ok := rawValue.(*networkservice.NetworkServiceServer)
	return value, ok
}

func storeServerRequestError(ctx context.Context, err *error) {
	metadata.Map(ctx, false).Store(serverRequestErrorKey, err)
}

func getServerRequestError(ctx context.Context) (err *error, ok bool) {
	return getError(ctx, false, serverRequestErrorKey)
}

func storeServerCloseError(ctx context.Context, err *error) {
	metadata.Map(ctx, false).Store(serverCloseErrorKey, err)
}

func getServerCloseError(ctx context.Context) (err *error, ok bool) {
	return getError(ctx, false, serverCloseErrorKey)
}

func storeClientRequestError(ctx context.Context, err *error) {
	metadata.Map(ctx, true).Store(clientRequestErrorKey, err)
}

func getClientRequestError(ctx context.Context) (err *error, ok bool) {
	return getError(ctx, true, clientRequestErrorKey)
}

func storeClientCloseError(ctx context.Context, err *error) {
	metadata.Map(ctx, true).Store(clientCloseErrorKey, err)
}

func getClientCloseError(ctx context.Context) (err *error, ok bool) {
	return getError(ctx, true, clientCloseErrorKey)
}

func withClientRequestTail(ctx context.Context, client *networkservice.NetworkServiceClient) {
	metadata.Map(ctx, true).Store(clientChainRequestTailKey, client)
}

func clientRequestTail(ctx context.Context) (client *networkservice.NetworkServiceClient, ok bool) {
	rawValue, ok := metadata.Map(ctx, true).Load(clientChainRequestTailKey)
	if !ok {
		return
	}
	value, ok := rawValue.(*networkservice.NetworkServiceClient)
	return value, ok
}

func withClientCloseTail(ctx context.Context, client *networkservice.NetworkServiceClient) {
	metadata.Map(ctx, true).Store(clientChainCloseTailKey, client)
}

func clientCloseTail(ctx context.Context) (client *networkservice.NetworkServiceClient, ok bool) {
	rawValue, ok := metadata.Map(ctx, true).Load(clientChainCloseTailKey)
	if !ok {
		return
	}
	value, ok := rawValue.(*networkservice.NetworkServiceClient)
	return value, ok
}

func isReadyForLogging(ctx context.Context, isClient bool) (isReady bool) {
	defer func() {
		if r := recover(); r != nil {
			isReady = false
		}
	}()
	metadata.Map(ctx, isClient)
	return true
}

func getError(ctx context.Context, isClient bool, key contextKeyType) (err *error, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).Load(key)
	if !ok {
		return
	}
	value, ok := rawValue.(*error)
	return value, ok
}
