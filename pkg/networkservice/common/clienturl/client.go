// Copyright (c) 2020-2021 Cisco Systems, Inc.
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

// Package clienturl provides server chain element that sets client URL in context
package clienturl

import (
	"context"
	"net/url"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type clientURLClient struct {
	u *url.URL
}

// NewClient - returns a new client chain element that sets client URL in context.
func NewClient(u *url.URL) networkservice.NetworkServiceClient {
	return &clientURLClient{u: u}
}

func (c *clientURLClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if c.u != nil {
		ctx = clienturlctx.WithClientURL(ctx, c.u)
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *clientURLClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if c.u != nil {
		ctx = clienturlctx.WithClientURL(ctx, c.u)
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
