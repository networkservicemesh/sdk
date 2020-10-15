// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020 Cisco Systems, Inc.
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

// Package next provides a mechanism for chained networkservice.NetworkService{Server,Client}s to call
// the next element in the chain.
package next

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

const (
	nextServerKey contextKeyType = "NextServer"
	nextClientKey contextKeyType = "NextClient"
)

type contextKeyType string

// withNextServer -
//    Wraps 'parent' in a new Context that has the Server networkservice.NetworkServiceServer to be called in the chain
//    Should only be set in CompositeEndpoint.Request/Close
func withNextServer(parent context.Context, next networkservice.NetworkServiceServer) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, nextServerKey, next)
}

// Server -
//   Returns the Server networkservice.NetworkServiceServer to be called in the chain from the context.Context
func Server(ctx context.Context) networkservice.NetworkServiceServer {
	rv, ok := ctx.Value(nextServerKey).(networkservice.NetworkServiceServer)
	if ok && rv != nil {
		return rv
	}
	return &tailServer{}
}

// withNextClient -
//    Wraps 'parent' in a new Context that has the Server networkservice.NetworkServiceServer to be called in the chain
//    Should only be set in CompositeEndpoint.Request/Close
func withNextClient(parent context.Context, next networkservice.NetworkServiceClient) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, nextClientKey, next)
}

// Client -
//   Returns the Client networkservice.NetworkServiceClient to be called in the chain from the context.Context
func Client(ctx context.Context) networkservice.NetworkServiceClient {
	rv, ok := ctx.Value(nextClientKey).(networkservice.NetworkServiceClient)
	if ok && rv != nil {
		return rv
	}
	return &tailClient{}
}
