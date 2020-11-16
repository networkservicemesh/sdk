// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package serialize_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

const (
	parallelCount = 1000
)

func TestSerializeServer_StressTest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.WarnLevel)
	defer logrus.SetLevel(logLevel)

	server := chain.NewNetworkServiceServer(
		serialize.NewServer(),
		new(requestServer),
		new(closeServer),
		newParallelServer(t),
		newDoubleCloseServer(t),
	)
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
		},
	}

	wg := new(sync.WaitGroup)
	wg.Add(parallelCount)
	for i := 0; i < parallelCount; i++ {
		go func() {
			defer wg.Done()
			conn, err := server.Request(ctx, request)
			assert.NoError(t, err)
			_, err = server.Close(ctx, conn)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}

type requestServer struct{}

func (s *requestServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	executor := serialize.RequestExecutor(ctx)
	go func() {
		executor.AsyncExec(func() error {
			_, err := next.Server(ctx).Request(serialize.WithExecutorsFromContext(context.TODO(), ctx), request)
			return err
		})
	}()

	return next.Server(ctx).Request(ctx, request)
}

func (s *requestServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

type closeServer struct{}

func (s *closeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	executor := serialize.CloseExecutor(ctx)
	go func() {
		executor.AsyncExec(func() error {
			_, err = next.Server(ctx).Close(context.TODO(), conn)
			return err
		})
	}()

	return conn, err
}

func (s *closeServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

type parallelServer struct {
	t     *testing.T
	state int32
}

func newParallelServer(t *testing.T) *parallelServer {
	return &parallelServer{
		t: t,
	}
}

func (s *parallelServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	state := atomic.LoadInt32(&s.state)
	assert.True(s.t, atomic.CompareAndSwapInt32(&s.state, state, state+1), "state has been changed")
	return next.Server(ctx).Request(ctx, request)
}

func (s *parallelServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	state := atomic.LoadInt32(&s.state)
	assert.True(s.t, atomic.CompareAndSwapInt32(&s.state, state, state+1), "state has been changed")
	return next.Server(ctx).Close(ctx, conn)
}

type doubleCloseServer struct {
	t           *testing.T
	lock        sync.Mutex
	connections map[string]bool
}

func newDoubleCloseServer(t *testing.T) *doubleCloseServer {
	return &doubleCloseServer{
		t:           t,
		connections: map[string]bool{},
	}
}

func (s *doubleCloseServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	s.lock.Lock()

	s.connections[request.GetConnection().GetId()] = true

	s.lock.Unlock()

	return next.Server(ctx).Request(ctx, request)
}

func (s *doubleCloseServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	s.lock.Lock()

	assert.True(s.t, s.connections[conn.GetId()], "closing not opened connection: %v", conn.GetId())
	s.connections[conn.GetId()] = false

	s.lock.Unlock()

	return next.Server(ctx).Close(ctx, conn)
}
