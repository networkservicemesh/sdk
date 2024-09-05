// Copyright (c) 2022 Cisco and/or its affiliates.
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
	"sync"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/tools/monitorconnection/next"
	"github.com/networkservicemesh/sdk/pkg/tools/monitorconnection/streamcontext"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
)

type testEmptyMCServer struct{}

func (t *testEmptyMCServer) MonitorConnections(in *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) error {
	return nil
}

type testEmptyMCMCServer struct {
	networkservice.MonitorConnection_MonitorConnectionsServer
	context context.Context
}

func (t *testEmptyMCMCServer) Send(event *networkservice.ConnectionEvent) error {
	return nil
}

func (t *testEmptyMCMCServer) Context() context.Context {
	return t.context
}

func emptyMCServer() networkservice.MonitorConnectionServer {
	return &testEmptyMCServer{}
}

type testVisitMCServer struct{}

func (t *testVisitMCServer) MonitorConnections(in *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) error {
	srv = streamcontext.MonitorConnectionMonitorConnectionsServer(visit(srv.Context()), srv)
	rv := next.MonitorConnectionServer(srv.Context()).MonitorConnections(in, srv)
	return rv
}

func visitMCServer() networkservice.MonitorConnectionServer {
	return &testVisitMCServer{}
}

func TestNewMonitorConnectionsServerShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_ = next.NewMonitorConnectionServer().MonitorConnections(
			nil, &testEmptyMCMCServer{context: context.Background()})
		_ = next.NewWrappedMonitorConnectionServer(func(server networkservice.MonitorConnectionServer) networkservice.MonitorConnectionServer {
			return server
		}).MonitorConnections(nil, &testEmptyMCMCServer{context: context.Background()})
	})
}

func TestNSServerBranches(t *testing.T) {
	servers := [][]networkservice.MonitorConnectionServer{
		{visitMCServer()},
		{visitMCServer(), visitMCServer()},
		{visitMCServer(), visitMCServer(), visitMCServer()},
		{emptyMCServer(), visitMCServer(), visitMCServer()},
		{visitMCServer(), emptyMCServer(), visitMCServer()},
		{visitMCServer(), visitMCServer(), emptyMCServer()},
		{next.NewMonitorConnectionServer(), next.NewMonitorConnectionServer(visitMCServer(), next.NewMonitorConnectionServer()), visitMCServer()},
	}
	expects := []int{1, 2, 3, 0, 1, 2, 2, 2}

	for i, sample := range servers {
		s := next.NewMonitorConnectionServer(sample...)
		ctx := visit(context.Background())
		eventSrv := &testEmptyMCMCServer{context: ctx}
		_ = s.MonitorConnections(nil, eventSrv)
		assert.Equal(t, expects[i], visitValue(eventSrv.Context()), fmt.Sprintf("sample index: %v", i))
	}
}

func TestDataRaceMonitorConnectionServer(t *testing.T) {
	s := next.NewMonitorConnectionServer(emptyMCServer())
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			_ = s.MonitorConnections(nil, &testEmptyMCMCServer{context: context.Background()})
		}()
	}

	wg.Wait()
}
