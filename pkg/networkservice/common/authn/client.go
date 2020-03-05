// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package authn

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/security"
)

const (
	defaultTokenExpiration = 5 * time.Minute
)

type authnClient struct {
	p security.Provider
}

// NewClient - returns a new authentication client
//			   p - provider of TLS certificate
func NewClient(p security.Provider) networkservice.NetworkServiceClient {
	return &authnClient{
		p: p,
	}
}

func (a *authnClient) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	token, err := security.GenerateToken(ctx, a.p, defaultTokenExpiration)
	if err != nil {
		return nil, err
	}

	in.GetConnection().GetPath().GetPathSegments()[in.GetConnection().GetPath().GetIndex()].Token = token
	return next.Client(ctx).Request(ctx, in, opts...)
}

func (a *authnClient) Close(ctx context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, in, opts...)
}
