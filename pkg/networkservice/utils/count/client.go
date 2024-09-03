// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
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
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// Client is a client type for counting Requests/Closes.
type Client struct {
	totalForwardRequests, totalForwardCloses   int32
	totalBackwardRequests, totalBackwardCloses int32
	forwardRequests, forwardCloses             map[string]int32
	backwardRequests, backwardCloses           map[string]int32
	forwardMu, backwardMu                      sync.Mutex
}

// Request performs request and increments requests count.
func (c *Client) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	/* Forward pass*/
	c.forwardMu.Lock()
	atomic.AddInt32(&c.totalForwardRequests, 1)
	if c.forwardRequests == nil {
		c.forwardRequests = make(map[string]int32)
	}
	c.forwardRequests[request.GetConnection().GetId()]++
	c.forwardMu.Unlock()

	/* Request */
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return conn, err
	}

	/* Backward pass*/
	c.backwardMu.Lock()
	atomic.AddInt32(&c.totalBackwardRequests, 1)
	if c.backwardRequests == nil {
		c.backwardRequests = make(map[string]int32)
	}
	c.backwardRequests[conn.GetId()]++
	c.backwardMu.Unlock()

	return conn, err
}

// Close performs close and increments closes count.
func (c *Client) Close(ctx context.Context, connection *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	/* Forward pass*/
	c.forwardMu.Lock()
	atomic.AddInt32(&c.totalForwardCloses, 1)
	if c.forwardCloses == nil {
		c.forwardCloses = make(map[string]int32)
	}
	c.forwardCloses[connection.GetId()]++
	c.forwardMu.Unlock()

	/* Close */
	r, err := next.Client(ctx).Close(ctx, connection, opts...)
	if err != nil {
		return r, err
	}

	/* Backward pass*/
	c.backwardMu.Lock()
	atomic.AddInt32(&c.totalBackwardCloses, 1)
	if c.backwardCloses == nil {
		c.backwardCloses = make(map[string]int32)
	}
	c.backwardCloses[connection.GetId()]++
	c.backwardMu.Unlock()

	return r, err
}

// Requests returns forward requests count.
func (c *Client) Requests() int {
	return int(atomic.LoadInt32(&c.totalForwardRequests))
}

// Closes returns forward closes count.
func (c *Client) Closes() int {
	return int(atomic.LoadInt32(&c.totalForwardCloses))
}

// BackwardRequests returns backward requests count.
func (c *Client) BackwardRequests() int {
	return int(atomic.LoadInt32(&c.totalBackwardRequests))
}

// BackwardCloses returns backward closes count.
func (c *Client) BackwardCloses() int {
	return int(atomic.LoadInt32(&c.totalBackwardCloses))
}

// UniqueRequests returns unique forward requests count.
func (c *Client) UniqueRequests() int {
	c.forwardMu.Lock()
	defer c.forwardMu.Unlock()

	if c.forwardRequests == nil {
		return 0
	}
	return len(c.forwardRequests)
}

// UniqueCloses returns unique forward closes count.
func (c *Client) UniqueCloses() int {
	c.forwardMu.Lock()
	defer c.forwardMu.Unlock()

	if c.forwardCloses == nil {
		return 0
	}
	return len(c.forwardCloses)
}

// UniqueBackwardRequests returns unique backward requests count.
func (c *Client) UniqueBackwardRequests() int {
	c.backwardMu.Lock()
	defer c.backwardMu.Unlock()

	if c.backwardRequests == nil {
		return 0
	}
	return len(c.backwardRequests)
}

// UniqueBackwardCloses returns unique backward closes count.
func (c *Client) UniqueBackwardCloses() int {
	c.backwardMu.Lock()
	defer c.backwardMu.Unlock()

	if c.backwardCloses == nil {
		return 0
	}
	return len(c.backwardCloses)
}
