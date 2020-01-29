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

package heal

import (
	"context"
	"github.com/gogo/protobuf/proto"
	"github.com/networkservicemesh/api/pkg/api/connection"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"sync"
	"testing"
	"time"
)

type testMonitor struct {
	sync.WaitGroup
	stopCh   chan struct{}
	streamCh chan connection.MonitorConnection_MonitorConnectionsServer
}

func newTestMonitor() *testMonitor {
	return &testMonitor{
		stopCh:   make(chan struct{}),
		streamCh: make(chan connection.MonitorConnection_MonitorConnectionsServer),
	}
}

func (t *testMonitor) MonitorConnections(s *connection.MonitorScopeSelector, stream connection.MonitorConnection_MonitorConnectionsServer) error {
	t.Add(1)
	defer t.Done()

	t.streamCh <- stream
	<-t.stopCh
	return nil
}

func (t *testMonitor) stream(ctx context.Context) (connection.MonitorConnection_MonitorConnectionsServer, func(), error) {
	closeFunc := func() {
		close(t.stopCh)
		t.Wait()
	}

	select {
	case s := <-t.streamCh:
		return s, closeFunc, nil
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}

func serve(s connection.MonitorConnectionServer) (*bufconn.Listener, func()) {
	srv := grpc.NewServer()
	connection.RegisterMonitorConnectionServer(srv, s)
	ln := bufconn.Listen(1024 * 1024)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = srv.Serve(ln)
	}()

	return ln, func() {
		_ = ln.Close()
		wg.Wait()
	}
}

func dial(ctx context.Context, ln *bufconn.Listener) (connection.MonitorConnectionClient, func(), error) {
	dialer := func(context context.Context, s string) (conn net.Conn, e error) {
		return ln.Dial()
	}

	cc, err := grpc.DialContext(ctx, "", grpc.WithInsecure(), grpc.WithContextDialer(dialer))
	if err != nil {
		return nil, nil, err
	}

	return connection.NewMonitorConnectionClient(cc), func() { _ = cc.Close() }, nil
}

func syncCondCheck(exec serialize.Executor, pred func() bool) func() bool {
	return func() bool {
		check := make(chan bool)
		exec.AsyncExec(func() {
			check <- pred()
		})
		return <-check
	}
}

func TestHealClient_New(t *testing.T) {
	suits := []struct {
		name     string
		events   []*connection.ConnectionEvent
		reported map[string]*connection.Connection
	}{
		{
			name: "initial state transfer",
			events: []*connection.ConnectionEvent{
				{
					Type: connection.ConnectionEventType_INITIAL_STATE_TRANSFER,
					Connections: map[string]*connection.Connection{
						"conn-1": {Id: "conn-1", NetworkService: "ns-1"},
						"conn-2": {Id: "conn-2", NetworkService: "ns-2"},
					},
				},
			},
			reported: map[string]*connection.Connection{
				"conn-1": {Id: "conn-1", NetworkService: "ns-1"},
				"conn-2": {Id: "conn-2", NetworkService: "ns-2"},
			},
		},
		{
			name: "update event",
			events: []*connection.ConnectionEvent{
				{
					Type: connection.ConnectionEventType_INITIAL_STATE_TRANSFER,
					Connections: map[string]*connection.Connection{
						"conn-1": {Id: "conn-1", NetworkService: "ns-1"},
						"conn-2": {Id: "conn-2", NetworkService: "ns-2"},
					},
				},
				{
					Type: connection.ConnectionEventType_UPDATE,
					Connections: map[string]*connection.Connection{
						"conn-1": {Id: "conn-1", NetworkService: "ns-1-upd"},
					},
				},
				{
					Type: connection.ConnectionEventType_UPDATE,
					Connections: map[string]*connection.Connection{
						"conn-2": {Id: "conn-2", NetworkService: "ns-2-upd"},
					},
				},
			},
			reported: map[string]*connection.Connection{
				"conn-1": {Id: "conn-1", NetworkService: "ns-1-upd"},
				"conn-2": {Id: "conn-2", NetworkService: "ns-2-upd"},
			},
		},
		{
			name: "delete event",
			events: []*connection.ConnectionEvent{
				{
					Type: connection.ConnectionEventType_INITIAL_STATE_TRANSFER,
					Connections: map[string]*connection.Connection{
						"conn-1": {Id: "conn-1", NetworkService: "ns-1"},
						"conn-2": {Id: "conn-2", NetworkService: "ns-2"},
					},
				},
				{
					Type: connection.ConnectionEventType_DELETE,
					Connections: map[string]*connection.Connection{
						"conn-1": {Id: "conn-1", NetworkService: "ns-1"},
					},
				},
			},
			reported: map[string]*connection.Connection{
				"conn-2": {Id: "conn-2", NetworkService: "ns-2"},
			},
		},
	}

	for i := range suits {
		s := suits[i]
		logrus.Info(s.name)
		ok := t.Run(s.name, func(t *testing.T) {
			tm := newTestMonitor()

			ln, cancelServe := serve(tm)
			defer cancelServe()

			m, cancelDial, err := dial(context.Background(), ln)
			require.Nil(t, err)
			defer cancelDial()

			healCl := NewClient(m, nil).(*healClient)
			require.True(t, healCl == *healCl.onHeal)
			defer healCl.Stop()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			stream, cancelStream, err := tm.stream(ctx)
			require.Nil(t, err)
			defer cancelStream()

			for _, event := range s.events {
				err = stream.Send(event)
				require.Nil(t, err)
			}

			cond := func() bool {
				equal := func(m1, m2 map[string]*connection.Connection) bool {
					for k, v := range m1 {
						if v2, ok := m2[k]; !ok || !proto.Equal(v, v2) {
							return false
						}
					}
					return true
				}

				return equal(healCl.reported, s.reported)
			}

			cond = syncCondCheck(healCl.updateExecutor, cond)
			require.Eventually(t, cond, 5*time.Second, 10*time.Millisecond)
		})
		logrus.Info(ok)
	}
}
