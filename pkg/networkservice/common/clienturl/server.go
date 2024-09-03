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

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type clientURLServer struct {
	u *url.URL
}

// NewServer - returns a new server chain element that sets client URL in context.
func NewServer(u *url.URL) networkservice.NetworkServiceServer {
	return &clientURLServer{u: u}
}

func (c *clientURLServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	ctx = clienturlctx.WithClientURL(ctx, c.u)
	return next.Server(ctx).Request(ctx, request)
}

func (c *clientURLServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	ctx = clienturlctx.WithClientURL(ctx, c.u)
	return next.Server(ctx).Close(ctx, conn)
}
