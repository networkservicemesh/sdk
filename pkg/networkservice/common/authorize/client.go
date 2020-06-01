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

package authorize

import (
	"context"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/security"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type authorizeClient struct {
	provider        security.Provider
	tokenExpiration time.Duration
}

const (
	defaultTokenExpiration = 15 * time.Minute
)

// NewClient - returns a new authorization networkservicemesh.NetworkServiceClient
func NewClient(provider security.Provider, options ...ClientOption) networkservice.NetworkServiceClient {
	c := &authorizeClient{
		provider: provider,
	}

	for _, o := range options {
		o.apply(c)
	}

	if c.tokenExpiration == 0 {
		c.tokenExpiration = defaultTokenExpiration
	}

	return c
}

func (a *authorizeClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	token, err := security.GenerateToken(ctx, a.provider, defaultTokenExpiration)
	if err != nil {
		return nil, err
	}
	index := request.GetConnection().GetPath().GetIndex()
	request.GetConnection().GetPath().GetPathSegments()[index].Token = token
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (a *authorizeClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	token, err := security.GenerateToken(ctx, a.provider, defaultTokenExpiration)
	if err != nil {
		return nil, err
	}
	index := conn.GetPath().GetIndex()
	conn.GetPath().GetPathSegments()[index].Token = token
	return next.Client(ctx).Close(ctx, conn, opts...)
}
