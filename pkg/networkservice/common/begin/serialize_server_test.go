// Copyright (c) 2021 Cisco and/or its affiliates.
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

package begin_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

const (
	parallelCount = 1000
)

func testRequest(id string) *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: id,
		},
	}
}

func TestSerializeServer_StressTest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := chain.NewNetworkServiceServer(
		begin.NewServer(),
		newParallelServer(t),
	)

	wg := new(sync.WaitGroup)
	wg.Add(parallelCount)
	for i := 0; i < parallelCount; i++ {
		go func(id string) {
			defer wg.Done()

			conn, err := server.Request(ctx, testRequest(id))
			assert.NoError(t, err)

			_, err = server.Close(ctx, conn)
			assert.NoError(t, err)
		}(fmt.Sprint(i % 20))
	}
	wg.Wait()
}

type parallelServer struct {
	t      *testing.T
	states sync.Map
}

func newParallelServer(t *testing.T) *parallelServer {
	return &parallelServer{
		t: t,
	}
}

func (s *parallelServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	raw, _ := s.states.LoadOrStore(request.GetConnection().GetId(), new(int32))
	statePtr := raw.(*int32)

	state := atomic.LoadInt32(statePtr)
	assert.True(s.t, atomic.CompareAndSwapInt32(statePtr, state, state+1), "state has been changed for connection %s expected %d actual %d", request.GetConnection().GetId(), state, atomic.LoadInt32(statePtr))

	return next.Server(ctx).Request(ctx, request)
}

func (s *parallelServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	raw, _ := s.states.LoadOrStore(conn.GetId(), new(int32))
	statePtr := raw.(*int32)

	state := atomic.LoadInt32(statePtr)
	assert.True(s.t, atomic.CompareAndSwapInt32(statePtr, state, state+1), "state has been changed for connection %s expected %d actual %d", conn.GetId(), state, atomic.LoadInt32(statePtr))

	return next.Server(ctx).Close(ctx, conn)
}
