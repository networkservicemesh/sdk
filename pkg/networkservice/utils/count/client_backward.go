// Copyright (c) 2024 Cisco and/or its affiliates.
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

package count

import (
	"context"
	"sync/atomic"

	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// ClientBackward checks the connection on the way back
type ClientBackward struct {
	Client
}

// Request performs request and increments requests count
func (c *ClientBackward) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return conn, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.AddInt32(&c.totalRequests, 1)
	if c.requests == nil {
		c.requests = make(map[string]int32)
	}
	c.requests[request.GetConnection().GetId()]++

	return conn, err
}

// Close performs close and increments closes count
func (c *ClientBackward) Close(ctx context.Context, connection *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	r, err := next.Client(ctx).Close(ctx, connection, opts...)
	if err != nil {
		return r, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.AddInt32(&c.totalCloses, 1)
	if c.closes == nil {
		c.closes = make(map[string]int32)
	}
	c.closes[connection.GetId()]++

	return r, err
}
