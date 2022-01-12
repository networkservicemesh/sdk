// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// Client is a client type for counting Requests/Closes
type Client struct {
	totalRequests, totalCloses int32
	requests, closes           map[string]int32
	mu                         sync.Mutex
}

// Request performs request and increments requests count
func (c *Client) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.AddInt32(&c.totalRequests, 1)
	if c.requests == nil {
		c.requests = make(map[string]int32)
	}
	c.requests[request.GetConnection().GetId()]++

	return next.Client(ctx).Request(ctx, request, opts...)
}

// Close performs close and increments closes count
func (c *Client) Close(ctx context.Context, connection *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.AddInt32(&c.totalCloses, 1)
	if c.closes == nil {
		c.closes = make(map[string]int32)
	}
	c.closes[connection.GetId()]++

	return next.Client(ctx).Close(ctx, connection, opts...)
}

// Requests returns requests count
func (c *Client) Requests() int {
	return int(atomic.LoadInt32(&c.totalRequests))
}

// Closes returns closes count
func (c *Client) Closes() int {
	return int(atomic.LoadInt32(&c.totalCloses))
}

// UniqueRequests returns unique requests count
func (c *Client) UniqueRequests() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.requests == nil {
		return 0
	}
	return len(c.requests)
}

// UniqueCloses returns unique closes count
func (c *Client) UniqueCloses() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closes == nil {
		return 0
	}
	return len(c.closes)
}
