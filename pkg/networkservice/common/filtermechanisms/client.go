// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2019-2021 VMware, Inc.
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

// Package filtermechanisms filters out remote mechanisms if communicating by remote url
// filters out local mechanisms otherwise.
package filtermechanisms

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
)

type filterMechanismsClient struct{}

// NewClient - filters out remote mechanisms if connection is received from a unix file socket, otherwise filters
// out local mechanisms.
func NewClient() networkservice.NetworkServiceClient {
	return new(filterMechanismsClient)
}

func (s *filterMechanismsClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	u := clienturlctx.ClientURL(ctx)
	if u.Scheme == "inode" || u.Scheme == "unix" {
		request.MechanismPreferences = filterMechanismsByCls(request.GetMechanismPreferences(), cls.LOCAL)
	} else {
		request.MechanismPreferences = filterMechanismsByCls(request.GetMechanismPreferences(), cls.REMOTE)
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (s *filterMechanismsClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func filterMechanismsByCls(mechanisms []*networkservice.Mechanism, mechanismCls string) []*networkservice.Mechanism {
	var result []*networkservice.Mechanism
	for _, mechanism := range mechanisms {
		if mechanism.GetCls() == mechanismCls {
			result = append(result, mechanism)
		}
	}
	return result
}
