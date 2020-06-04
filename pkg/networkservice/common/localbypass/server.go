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

// Package localbypass provides a NetworkServiceServer chain element that tracks local Endpoints and substitutes
// their unix file socket as the clienturl.ClientURL(ctx) used to connect to them.
package localbypass

import (
	"context"
	"net/url"

	"github.com/networkservicemesh/sdk/pkg/registry/memory"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type localBypassServer struct {
	storage *memory.Storage
}

// NewServer - creates a NetworkServiceServer that tracks locally registered Endpoints substitutes their
//             passed endpoint_address with clienturl.ClientURL(ctx) used to connect to them.
func NewServer(storage *memory.Storage) networkservice.NetworkServiceServer {
	rv := &localBypassServer{
		storage: storage,
	}
	return rv
}

func (l *localBypassServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if endpoint, ok := l.storage.NetworkServiceEndpoints.Load(request.GetConnection().GetNetworkServiceEndpointName()); ok && endpoint != nil {
		if u, err := url.Parse(endpoint.Url); err == nil {
			ctx = clienturl.WithClientURL(ctx, u)
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (l *localBypassServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if endpoint, ok := l.storage.NetworkServiceEndpoints.Load(conn.GetNetworkServiceEndpointName()); ok && endpoint != nil {
		if u, err := url.Parse(endpoint.Url); err == nil {
			ctx = clienturl.WithClientURL(ctx, u)
		}
	}
	return next.Server(ctx).Close(ctx, conn)
}
