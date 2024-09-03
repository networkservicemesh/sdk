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

package injectexcludedprefixes

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type injectExcludedPrefixesServer struct {
	prefixes []string
}

func (ieps *injectExcludedPrefixesServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn := request.GetConnection()
	if conn.GetContext() == nil {
		conn.Context = &networkservice.ConnectionContext{}
	}
	if conn.GetContext().GetIpContext() == nil {
		conn.Context.IpContext = &networkservice.IPContext{}
	}

	ipCtx := conn.GetContext().GetIpContext()
	ipCtx.ExcludedPrefixes = ieps.prefixes

	return next.Server(ctx).Request(ctx, request)
}

func (ieps *injectExcludedPrefixesServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, connection)
}

// NewServer - creates a networkservice.NetworkServiceServer chain element injecting specified excluded prefixes on Request into IP context.
func NewServer(excludedPrefixes []string) networkservice.NetworkServiceServer {
	return &injectExcludedPrefixesServer{
		prefixes: excludedPrefixes,
	}
}
