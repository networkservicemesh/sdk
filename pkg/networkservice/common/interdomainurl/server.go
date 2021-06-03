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

// Package interdomainurl inject into context clienturl if requested networkserviceendpoint has interdomain attribute
package interdomainurl

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/common/storeurl"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
	"github.com/networkservicemesh/sdk/pkg/tools/stringurl"
)

type interdomainurlServer struct {
	m stringurl.Map
}

// NewServer - returns a new NetworkServiceServer that injects clienturl into context on requesting known endpoint
func NewServer(rs *registry.NetworkServiceEndpointRegistryServer) networkservice.NetworkServiceServer {
	var rv = new(interdomainurlServer)
	*rs = storeurl.NewNetworkServiceEndpointRegistryServer(&rv.m)
	return rv
}

func (n *interdomainurlServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	u, ok := n.m.Load(request.Connection.NetworkServiceEndpointName)
	if ok {
		ctx = clienturlctx.WithClientURL(ctx, u)
		originalNSEName := request.GetConnection().NetworkServiceEndpointName
		request.GetConnection().NetworkServiceEndpointName = interdomain.Target(originalNSEName)
		resp, err := next.Server(ctx).Request(ctx, request)
		if err != nil {
			return nil, err
		}
		resp.NetworkServiceEndpointName = originalNSEName
		return resp, nil
	}

	return next.Server(ctx).Request(ctx, request)
}

func (n *interdomainurlServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	u, ok := n.m.Load(conn.NetworkServiceEndpointName)
	if ok {
		ctx = clienturlctx.WithClientURL(ctx, u)
		originalNSEName := conn.NetworkServiceEndpointName
		defer func() {
			conn.NetworkServiceEndpointName = originalNSEName
		}()
		conn.NetworkServiceEndpointName = interdomain.Target(originalNSEName)
	}
	return next.Server(ctx).Close(ctx, conn)
}
