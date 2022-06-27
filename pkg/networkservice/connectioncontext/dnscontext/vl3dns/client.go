// Copyright (c) 2022 Cisco and/or its affiliates.
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

package vl3dns

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type vl3DNSClient struct {
	listenOn url.URL
}

// NewClient - returns a new null client that does nothing but call next.Client(ctx).{Request/Close} and return the result
//             This is very useful in testing
func NewClient(listenOn url.URL) networkservice.NetworkServiceClient {
	return &vl3DNSClient{
		listenOn: listenOn,
	}
}

func (n *vl3DNSClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if request.GetConnection() == nil {
		request.Connection = new(networkservice.Connection)
	}
	if request.GetConnection().GetContext() == nil {
		request.GetConnection().Context = new(networkservice.ConnectionContext)
	}
	if request.GetConnection().GetContext().GetDnsContext() == nil {
		request.GetConnection().GetContext().DnsContext = new(networkservice.DNSContext)
	}

	request.GetConnection().GetContext().GetDnsContext().Configs = []*networkservice.DNSConfig{
		{
			DnsServerIps: []string{n.listenOn.Hostname()},
		},
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (n *vl3DNSClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
