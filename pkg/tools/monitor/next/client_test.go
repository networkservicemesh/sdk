// Copyright (c) 2022 Doc.ai and/or its affiliates.
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

package next_test

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/monitor/next"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
)

type testEmptyMCMCClient struct {
	ctx context.Context
	grpc.ClientStream
}

func (t *testEmptyMCMCClient) Recv() (*networkservice.ConnectionEvent, error) {
	return nil, io.EOF
}

func (t *testEmptyMCMCClient) Context() context.Context {
	return t.ctx
}

type testEmptyMCClient struct{}

func (t *testEmptyMCClient) MonitorConnections(ctx context.Context, in *networkservice.MonitorScopeSelector, opts ...grpc.CallOption) (networkservice.MonitorConnection_MonitorConnectionsClient, error) {
	return &testEmptyMCMCClient{ctx: ctx}, nil
}
func emptyMCClient() networkservice.MonitorConnectionClient {
	return &testEmptyMCClient{}
}

type testVisitMCClient struct{}

// MonitorConnections(ctx context.Context, in *MonitorScopeSelector, opts ...grpc.CallOption) (MonitorConnection_MonitorConnectionsClient, error)
func (t *testVisitMCClient) MonitorConnections(ctx context.Context, in *networkservice.MonitorScopeSelector, opts ...grpc.CallOption) (networkservice.MonitorConnection_MonitorConnectionsClient, error) {
	return next.MonitorConnectionClient(ctx).MonitorConnections(visit(ctx), in, opts...)
}

func visitMCClient() networkservice.MonitorConnectionClient {
	return &testVisitMCClient{}
}

func TestNewMonitorConnectionClientShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_, _ = next.NewMonitorConnectionClient().MonitorConnections(context.Background(), nil)
		_, _ = next.NewWrappedMonitorConnectionClient(func(client networkservice.MonitorConnectionClient) networkservice.MonitorConnectionClient {
			return client
		}).MonitorConnections(context.Background(), nil)
	})
}

func TestDataRaceMonitorConnectionClient(t *testing.T) {
	c := next.NewMonitorConnectionClient(emptyMCClient())
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.MonitorConnections(context.Background(), nil)
		}()
	}
	wg.Wait()
}

func TestNSClientBranches(t *testing.T) {
	servers := [][]networkservice.MonitorConnectionClient{
		{visitMCClient()},
		{visitMCClient(), visitMCClient()},
		{visitMCClient(), visitMCClient(), visitMCClient()},
		{emptyMCClient(), visitMCClient(), visitMCClient()},
		{visitMCClient(), emptyMCClient(), visitMCClient()},
		{visitMCClient(), visitMCClient(), emptyMCClient()},
		{next.NewMonitorConnectionClient(), next.NewMonitorConnectionClient(visitMCClient(), next.NewMonitorConnectionClient()), visitMCClient()},
	}
	expects := []int{1, 2, 3, 0, 1, 2, 2, 2}
	for i, sample := range servers {
		s := next.NewMonitorConnectionClient(sample...)
		ctx := visit(context.Background())
		_, _ = s.MonitorConnections(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))
	}
}