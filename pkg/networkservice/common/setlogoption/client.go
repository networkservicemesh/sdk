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

package setlogoption

import (
	"context"

	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type setLogOptionClient struct {
	options map[string]string
}

// NewClient - construct a new set log option server to override some logging capabilities for context.
func NewClient(options map[string]string) networkservice.NetworkServiceClient {
	return &setLogOptionClient{
		options: options,
	}
}

func (s *setLogOptionClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	ctx = s.withFields(ctx)
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (s *setLogOptionClient) withFields(ctx context.Context) context.Context {
	ctxFields := log.Fields(ctx)
	fields := make(map[string]interface{})
	for k, v := range ctxFields {
		fields[k] = v
	}

	fields["type"] = networkService
	for k, v := range s.options {
		fields[k] = v
	}
	if len(fields) > 0 {
		ctx = log.WithFields(ctx, fields)
	}
	return ctx
}

func (s *setLogOptionClient) Close(ctx context.Context, connection *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	ctx = s.withFields(ctx)
	return next.Client(ctx).Close(ctx, connection, opts...)
}
