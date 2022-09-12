// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

// Package checkconnection provides utilities for checking the returned connection from the next element in a chain
package checkconnection

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type checkConnectionOnReturn struct {
	*testing.T
	check func(t *testing.T, connection *networkservice.Connection)
}

// NewClient - returns a NetworkServiceClient chain element that will check the connection returned by the next Client in the chain
//
//	t - *testing.T for the check
//	check - function to run on the Connection to check it is as it should be
func NewClient(t *testing.T, check func(t *testing.T, conn *networkservice.Connection)) networkservice.NetworkServiceClient {
	return &checkConnectionOnReturn{
		T:     t,
		check: check,
	}
}

func (c *checkConnectionOnReturn) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	c.check(c.T, conn)
	return conn, err
}

func (c *checkConnectionOnReturn) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn)
}
