// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package nsmgr_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func defaultRegistryService() *registry.NetworkService {
	return &registry.NetworkService{
		Name: "ns-" + uuid.New().String(),
	}
}

func defaultRegistryEndpoint(nsName string) *registry.NetworkServiceEndpoint {
	return &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{nsName},
	}
}

func defaultRequest(nsName string) *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: nsName,
			Context:        &networkservice.ConnectionContext{},
		},
	}
}

func testNSEAndClient(
	ctx context.Context,
	t *testing.T,
	domain *sandbox.Domain,
	nseReg *registry.NetworkServiceEndpoint,
) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)
	require.NoError(t, err)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest(nseReg.NetworkServiceNames[0])

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)

	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	_, err = domain.Nodes[0].EndpointRegistryClient.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

type passThroughClient struct {
	networkService string
}

func newPassTroughClient(networkService string) *passThroughClient {
	return &passThroughClient{
		networkService: networkService,
	}
}

func (c *passThroughClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	request.Connection.NetworkService = c.networkService
	request.Connection.NetworkServiceEndpointName = ""
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *passThroughClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	conn.NetworkService = c.networkService
	return next.Client(ctx).Close(ctx, conn, opts...)
}

type counterServer struct {
	Requests, Closes int32
	requests         map[string]int32
	closes           map[string]int32
	mu               sync.Mutex
}

func (c *counterServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.AddInt32(&c.Requests, 1)
	if c.requests == nil {
		c.requests = make(map[string]int32)
	}
	c.requests[request.GetConnection().GetId()]++

	return next.Server(ctx).Request(ctx, request)
}

func (c *counterServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.AddInt32(&c.Closes, 1)
	if c.closes == nil {
		c.closes = make(map[string]int32)
	}
	c.closes[connection.GetId()]++

	return next.Server(ctx).Close(ctx, connection)
}

func (c *counterServer) UniqueRequests() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.requests == nil {
		return 0
	}
	return len(c.requests)
}

func (c *counterServer) UniqueCloses() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closes == nil {
		return 0
	}
	return len(c.closes)
}
