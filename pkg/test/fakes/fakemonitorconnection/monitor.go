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

// fakemonitorconnection contains implementation of fake monitor server for tests
package fakemonitorconnection

import (
	"context"
	"net"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type fakeMonitorServer struct {
	stopCh   chan struct{}
	streamCh chan networkservice.MonitorConnection_MonitorConnectionsServer

	ln         *bufconn.Listener
	closeFuncs []func()
}

// New creates instance of fakeMonitorServer
func New() *fakeMonitorServer {
	rv := &fakeMonitorServer{
		stopCh:   make(chan struct{}),
		streamCh: make(chan networkservice.MonitorConnection_MonitorConnectionsServer),
	}
	rv.serve()
	return rv
}

func (f *fakeMonitorServer) MonitorConnections(s *networkservice.MonitorScopeSelector, stream networkservice.MonitorConnection_MonitorConnectionsServer) error {
	f.streamCh <- stream
	<-f.stopCh
	return nil
}

func (f *fakeMonitorServer) Stream(ctx context.Context) (networkservice.MonitorConnection_MonitorConnectionsServer, func(), error) {
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

func (f *fakeMonitorServer) Client(ctx context.Context) (networkservice.MonitorConnectionClient, error) {
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

func (f *fakeMonitorServer) serve() {
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

func (f *fakeMonitorServer) pushOnCloseFunc(h func()) {
	f.closeFuncs = append([]func(){h}, f.closeFuncs...)
}

func (f *fakeMonitorServer) Close() {
	for _, c := range f.closeFuncs {
		c()
	}
}
