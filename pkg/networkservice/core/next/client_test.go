// Copyright (c) 2020 Doc.ai, Inc.
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

package next

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/connection"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type testNetworkServiceClient struct {
	request func(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error)
	close   func(ctx context.Context, in *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error)
}

func (t *testNetworkServiceClient) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	if t != nil {
		return t.request(ctx, in)
	}
	return nil, nil
}
func (t *testNetworkServiceClient) Close(ctx context.Context, in *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if t != nil {
		return t.close(ctx, in)
	}
	return nil, nil
}

func TestNewNetworkServiceClientShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		NewNetworkServiceClient(&testNetworkServiceClient{})
	})
}

func TestClientBranches(t *testing.T) {
	var visit []bool
	index := 0
	next := &testNetworkServiceClient{
		request: func(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
			visit[index] = !visit[index]
			index++
			return Client(ctx).Request(ctx, request)
		},
		close: func(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
			visit[index] = !visit[index]
			index++
			return Client(ctx).Close(ctx, conn)
		},
	}
	breaker := &testNetworkServiceClient{
		request: func(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
			visit[index] = !visit[index]
			index++
			return nil, nil
		},
		close: func(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
			visit[index] = !visit[index]
			index++
			return nil, nil
		},
	}
	samples := [][]networkservice.NetworkServiceClient{
		{next},
		{next, next},
		{next, next, next},
		{breaker, next, next},
		{next, breaker, next},
		{next, next, breaker},
	}
	expects := [][]bool{
		{true},
		{true, true},
		{true, true, true},
		{true, false, false},
		{true, true, false},
		{true, true, true},
	}
	reset := func(count int) {
		visit = make([]bool, count)
		index = 0
	}
	for i, sample := range samples {
		for _, ctx := range []context.Context{nil, context.Background()} {
			count := len(sample)
			reset(count)
			expect := expects[i]
			c := NewNetworkServiceClient(sample...)
			_, _ = c.Request(ctx, nil)
			assert.Equal(t, expect, visit)
			reset(count)
			_, _ = c.Close(ctx, nil)
			assert.Equal(t, expect, visit)
		}
	}
}
