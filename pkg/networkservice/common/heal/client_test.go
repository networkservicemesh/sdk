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
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
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
	streamCh chan networkservice.MonitorConnection_MonitorConnectionsServer
}

func newTestMonitor() *testMonitor {
	return &testMonitor{
		stopCh:   make(chan struct{}),
		streamCh: make(chan networkservice.MonitorConnection_MonitorConnectionsServer),
	}
}

func (t *testMonitor) MonitorConnections(s *networkservice.MonitorScopeSelector, stream networkservice.MonitorConnection_MonitorConnectionsServer) error {
	t.Add(1)
	defer t.Done()

	t.streamCh <- stream
	<-t.stopCh
	return nil
}

func (t *testMonitor) stream(ctx context.Context) (networkservice.MonitorConnection_MonitorConnectionsServer, func(), error) {
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

func serve(s networkservice.MonitorConnectionServer) (*bufconn.Listener, func()) {
	srv := grpc.NewServer()
	networkservice.RegisterMonitorConnectionServer(srv, s)
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

func dial(ctx context.Context, ln *bufconn.Listener) (networkservice.MonitorConnectionClient, func(), error) {
	dialer := func(context context.Context, s string) (conn net.Conn, e error) {
		return ln.Dial()
	}

	cc, err := grpc.DialContext(ctx, "", grpc.WithInsecure(), grpc.WithContextDialer(dialer))
	if err != nil {
		return nil, nil, err
	}

	return networkservice.NewMonitorConnectionClient(cc), func() { _ = cc.Close() }, nil
}

func syncCond(exec serialize.Executor, cond func() bool) func() bool {
	return func() bool {
		check := make(chan bool)
		exec.AsyncExec(func() {
			check <- cond()
		})
		return <-check
	}
}

type fixture struct {
	tm           *testMonitor
	closeFuncs   []func()
	serverStream networkservice.MonitorConnection_MonitorConnectionsServer
	client       *healClient
	onHeal       networkservice.NetworkServiceClient
}

func newFixture() (*fixture, error) {
	rv := &fixture{
		tm: newTestMonitor(),
	}

	ln, cancelServe := serve(rv.tm)
	rv.pushOnCloseFunc(cancelServe)

	m, cancelDial, err := dial(context.Background(), ln)
	if err != nil {
		return nil, err
	}
	rv.pushOnCloseFunc(cancelDial)

	rv.onHeal = &testOnHeal{}
	rv.client = NewClient(m, &rv.onHeal).(*healClient)
	rv.pushOnCloseFunc(rv.client.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	rv.pushOnCloseFunc(cancel)

	serverStream, cancelStream, err := rv.tm.stream(ctx)
	if err != nil {
		return nil, err
	}
	rv.serverStream = serverStream
	rv.pushOnCloseFunc(cancelStream)
	return rv, nil
}

func (f *fixture) pushOnCloseFunc(h func()) {
	f.closeFuncs = append([]func(){h}, f.closeFuncs...)
}

func (f *fixture) close() {
	for _, c := range f.closeFuncs {
		c()
	}
}

type clientRequest func(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error)

type testOnHeal struct {
	r clientRequest
}

func (t *testOnHeal) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	return t.r(ctx, in, opts...)
}

func (t *testOnHeal) Close(ctx context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	panic("implement me")
}

// calledOnce closes notifier channel when h called once
func calledOnce(notifier chan struct{}, h clientRequest) clientRequest {
	return func(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (i *networkservice.Connection, e error) {
		close(notifier)
		if h != nil {
			return h(ctx, in, opts...)
		}
		return nil, nil
	}
}

func TestHealClient_New(t *testing.T) {
	suits := []struct {
		name     string
		events   []*networkservice.ConnectionEvent
		reported map[string]*networkservice.Connection
	}{
		{
			name: "initial state transfer",
			events: []*networkservice.ConnectionEvent{
				{
					Type: networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
					Connections: map[string]*networkservice.Connection{
						"conn-1": {Id: "conn-1", NetworkService: "ns-1"},
						"conn-2": {Id: "conn-2", NetworkService: "ns-2"},
					},
				},
			},
			reported: map[string]*networkservice.Connection{
				"conn-1": {Id: "conn-1", NetworkService: "ns-1"},
				"conn-2": {Id: "conn-2", NetworkService: "ns-2"},
			},
		},
		{
			name: "update event",
			events: []*networkservice.ConnectionEvent{
				{
					Type: networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
					Connections: map[string]*networkservice.Connection{
						"conn-1": {Id: "conn-1", NetworkService: "ns-1"},
						"conn-2": {Id: "conn-2", NetworkService: "ns-2"},
					},
				},
				{
					Type: networkservice.ConnectionEventType_UPDATE,
					Connections: map[string]*networkservice.Connection{
						"conn-1": {Id: "conn-1", NetworkService: "ns-1-upd"},
					},
				},
				{
					Type: networkservice.ConnectionEventType_UPDATE,
					Connections: map[string]*networkservice.Connection{
						"conn-2": {Id: "conn-2", NetworkService: "ns-2-upd"},
					},
				},
			},
			reported: map[string]*networkservice.Connection{
				"conn-1": {Id: "conn-1", NetworkService: "ns-1-upd"},
				"conn-2": {Id: "conn-2", NetworkService: "ns-2-upd"},
			},
		},
		{
			name: "delete event",
			events: []*networkservice.ConnectionEvent{
				{
					Type: networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
					Connections: map[string]*networkservice.Connection{
						"conn-1": {Id: "conn-1", NetworkService: "ns-1"},
						"conn-2": {Id: "conn-2", NetworkService: "ns-2"},
					},
				},
				{
					Type: networkservice.ConnectionEventType_DELETE,
					Connections: map[string]*networkservice.Connection{
						"conn-1": {Id: "conn-1", NetworkService: "ns-1"},
					},
				},
			},
			reported: map[string]*networkservice.Connection{
				"conn-2": {Id: "conn-2", NetworkService: "ns-2"},
			},
		},
	}

	for i := range suits {
		s := suits[i]
		logrus.Info(s.name)
		ok := t.Run(s.name, func(t *testing.T) {
			f, err := newFixture()
			require.Nil(t, err)
			defer f.close()

			for _, event := range s.events {
				err = f.serverStream.Send(event)
				require.Nil(t, err)
			}

			cond := func() bool {
				equal := func(m1, m2 map[string]*networkservice.Connection) bool {
					for k, v := range m1 {
						if v2, ok := m2[k]; !ok || !proto.Equal(v, v2) {
							return false
						}
					}
					return true
				}

				return equal(f.client.reported, s.reported)
			}

			cond = syncCond(f.client.updateExecutor, cond)
			require.Eventually(t, cond, 5*time.Second, 10*time.Millisecond)
		})
		logrus.Info(ok)
	}
}

func TestHealClient_Request(t *testing.T) {
	f, err := newFixture()
	require.Nil(t, err)
	healChain := chain.NewNetworkServiceClient(f.client)

	conn, err := healChain.Request(context.Background(), &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             "conn-1",
			NetworkService: "ns-1",
		},
	})

	cond := func() bool {
		_, okR := f.client.requestors[conn.GetId()]
		_, okC := f.client.closers[conn.GetId()]
		return okR && okC
	}
	cond = syncCond(f.client.updateExecutor, cond)
	require.True(t, cond())

	calledOnceNotifier := make(chan struct{})
	f.onHeal.(*testOnHeal).r = calledOnce(calledOnceNotifier, f.onHeal.(*testOnHeal).r)

	err = f.serverStream.Send(&networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
		Connections: map[string]*networkservice.Connection{
			"conn-1": {
				Id:             "conn-1",
				NetworkService: "ns-1",
			},
		},
	})
	require.Nil(t, err)

	err = f.serverStream.Send(&networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_DELETE,
		Connections: map[string]*networkservice.Connection{
			"conn-1": {
				Id:             "conn-1",
				NetworkService: "ns-1",
			},
		},
	})
	require.Nil(t, err)

	cond = func() bool {
		select {
		case <-calledOnceNotifier:
			return true
		default:
			return false
		}
	}
	require.Eventually(t, cond, 5*time.Second, 10*time.Millisecond)
}

func TestHealClient_Close(t *testing.T) {
	f, err := newFixture()
	require.Nil(t, err)
	healChain := chain.NewNetworkServiceClient(f.client)

	conn, err := healChain.Request(context.Background(), &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             "conn-1",
			NetworkService: "ns-1",
		},
	})

	cond := func() bool {
		_, okR := f.client.requestors[conn.GetId()]
		_, okC := f.client.closers[conn.GetId()]
		return okR && okC
	}
	cond = syncCond(f.client.updateExecutor, cond)
	require.True(t, cond())

	_, err = healChain.Close(context.Background(), &networkservice.Connection{Id: "conn-1"})
	require.Nil(t, err)

	cond = func() bool {
		_, okR := f.client.requestors[conn.GetId()]
		_, okC := f.client.closers[conn.GetId()]
		return !okR && !okC
	}
	cond = syncCond(f.client.updateExecutor, cond)
	require.True(t, cond())
}
