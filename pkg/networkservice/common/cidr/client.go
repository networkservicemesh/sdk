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

package cidr

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type cidrClient struct {
	prefixLen uint32
	family    networkservice.IpFamily_Family
}

// NewClient creates a NetworkServiceClient chain element that requests ExtraPrefix
func NewClient(prefixLen uint32, family networkservice.IpFamily_Family) networkservice.NetworkServiceClient {
	return &cidrClient{
		prefixLen: prefixLen,
		family:    family,
	}
}

func (c *cidrClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn := request.GetConnection()
	if conn.GetContext() == nil {
		conn.Context = &networkservice.ConnectionContext{}
	}
	if conn.GetContext().GetIpContext() == nil {
		conn.Context.IpContext = &networkservice.IPContext{}
	}
	ipContext := conn.GetContext().GetIpContext()

	// Add ExtraPrefixRequest if there is no extra prefix
	if ipContext.GetExtraPrefixes() == nil {
		ipContext.ExtraPrefixRequest = append(ipContext.ExtraPrefixRequest, &networkservice.ExtraPrefixRequest{
			AddrFamily:      &networkservice.IpFamily{Family: c.family},
			PrefixLen:       c.prefixLen,
			RequiredNumber:  1,
			RequestedNumber: 1,
		})
	}

	// Add ExtraPrefix to the route for the remote side
	loaded := load(ctx, metadata.IsClient(c))
	if !loaded && ipContext.GetExtraPrefixes() != nil {
		for _, item := range ipContext.ExtraPrefixes {
			ipContext.DstRoutes = append(ipContext.DstRoutes, &networkservice.Route{
				Prefix: item,
			})
		}
		store(ctx, metadata.IsClient(c))
	}

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil && !loaded {
		delete(ctx, metadata.IsClient(c))
	}

	return conn, err
}

func (c *cidrClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	delete(ctx, metadata.IsClient(c))
	return next.Client(ctx).Close(ctx, conn, opts...)
}
