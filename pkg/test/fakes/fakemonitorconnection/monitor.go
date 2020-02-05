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

// Package fakemonitorconnection contains implementation of fake monitor server for tests
package fakemonitorconnection

import (
	"context"
	"net"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// FakeMonitorServer implements networkservice.MonitorConnectionServer for test
type FakeMonitorServer struct {
	stopCh   chan struct{}
	streamCh chan networkservice.MonitorConnection_MonitorConnectionsServer

	ln         *bufconn.Listener
	closeFuncs []func()
}

// New creates instance of FakeMonitorServer
func New() *FakeMonitorServer {
	rv := &FakeMonitorServer{
		stopCh:   make(chan struct{}),
		streamCh: make(chan networkservice.MonitorConnection_MonitorConnectionsServer),
	}
	rv.serve()
	return rv
}

// MonitorConnections is a fake implementation of MonitorConnections, pushes 'stream' to channel
// that lately could be obtained using method Stream
func (f *FakeMonitorServer) MonitorConnections(s *networkservice.MonitorScopeSelector, stream networkservice.MonitorConnection_MonitorConnectionsServer) error {
	f.streamCh <- stream
	<-f.stopCh
	return nil
}

// Stream returns the last server 'stream' with blocking
func (f *FakeMonitorServer) Stream(ctx context.Context) (networkservice.MonitorConnection_MonitorConnectionsServer, func(), error) {
	closeFunc := func() {
		close(f.stopCh)
	}

	select {
	case s := <-f.streamCh:
		return s, closeFunc, nil
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}

// Client returns client for fake monitor server
func (f *FakeMonitorServer) Client(ctx context.Context) (networkservice.MonitorConnectionClient, error) {
	dialer := func(context context.Context, s string) (conn net.Conn, e error) {
		return f.ln.Dial()
	}

	cc, err := grpc.DialContext(ctx, "", grpc.WithInsecure(), grpc.WithContextDialer(dialer))
	if err != nil {
		return nil, err
	}

	f.pushOnCloseFunc(func() { _ = cc.Close() })

	return networkservice.NewMonitorConnectionClient(cc), nil
}

// Close releases all resources
func (f *FakeMonitorServer) Close() {
	for _, c := range f.closeFuncs {
		c()
	}
}

func (f *FakeMonitorServer) serve() {
	srv := grpc.NewServer()
	networkservice.RegisterMonitorConnectionServer(srv, f)
	f.ln = bufconn.Listen(1024 * 1024)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = srv.Serve(f.ln)
	}()

	f.pushOnCloseFunc(func() {
		_ = f.ln.Close()
		wg.Wait()
	})
}

func (f *FakeMonitorServer) pushOnCloseFunc(h func()) {
	f.closeFuncs = append([]func(){h}, f.closeFuncs...)
}
