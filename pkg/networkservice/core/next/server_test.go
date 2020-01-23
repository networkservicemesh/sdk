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

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
)

type testNetworkServiceServer struct {
	request func(context.Context, *networkservice.NetworkServiceRequest) (*connection.Connection, error)
	close   func(context.Context, *connection.Connection) (*empty.Empty, error)
}

func (t *testNetworkServiceServer) Request(ctx context.Context, r *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if t != nil {
		return t.request(ctx, r)
	}
	return nil, nil
}

func (t *testNetworkServiceServer) Close(ctx context.Context, r *connection.Connection) (*empty.Empty, error) {
	if t != nil {
		return t.close(ctx, r)
	}
	return nil, nil
}

func TestNewNetworkServiceServerShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		NewNetworkServiceServer(&testNetworkServiceServer{})
	})
}

func TestServerBranches(t *testing.T) {
	var visit []bool
	index := 0
	next := &testNetworkServiceServer{
		request: func(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
			visit[index] = !visit[index]
			index++
			return Server(ctx).Request(ctx, request)
		},
		close: func(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
			visit[index] = !visit[index]
			index++
			return Server(ctx).Close(ctx, conn)
		},
	}
	breaker := &testNetworkServiceServer{
		request: func(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
			visit[index] = !visit[index]
			index++
			return nil, nil
		},
		close: func(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
			visit[index] = !visit[index]
			index++
			return nil, nil
		},
	}
	samples := [][]networkservice.NetworkServiceServer{
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
		for _, ctx := range []context.Context{context.Background(), nil} {
			count := len(sample)
			reset(count)
			expect := expects[i]
			s := NewNetworkServiceServer(sample...)
			_, _ = s.Request(ctx, nil)
			assert.Equal(t, expect, visit)
			reset(count)
			_, _ = s.Close(ctx, nil)
			assert.Equal(t, expect, visit)
		}
	}
}
