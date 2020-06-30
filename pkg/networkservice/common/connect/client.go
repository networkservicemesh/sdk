// Copyright (c) 2020 Cisco and/or its affiliates.
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

package connect

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type connectClient struct {
	connection *networkservice.Connection
	cancel     context.CancelFunc
	mu         sync.Mutex
}

// NewClient - client chain element for use with single incoming connection, translates from incoming server connection to
//             outgoing client connection
func NewClient(cancel context.CancelFunc) networkservice.NetworkServiceClient {
	return &connectClient{
		cancel: cancel,
	}
}

func (c *connectClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	clientRequest := &networkservice.NetworkServiceRequest{Connection: request.GetConnection().Clone()}
	clientRequest.GetConnection().Mechanism = nil
	if c.connection != nil {
		clientRequest.Connection = c.connection
	}
	conn, err := next.Client(ctx).Request(ctx, clientRequest)
	c.connection = conn

	return conn, err
}

func (c *connectClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.connection == nil {
		return &empty.Empty{}, nil
	}
	_, err := next.Client(ctx).Close(ctx, c.connection)
	c.cancel()
	return &empty.Empty{}, err
}
