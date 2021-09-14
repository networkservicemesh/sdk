// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package endpoint

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type combineMonitorServer struct {
	monitorServers map[networkservice.MonitorConnectionServer]int
}

func (m *combineMonitorServer) MonitorConnections(selector *networkservice.MonitorScopeSelector, rawSrv networkservice.MonitorConnection_MonitorConnectionsServer) error {
	ctx, cancel := context.WithCancel(rawSrv.Context())
	defer cancel()

	var initChs []chan *networkservice.ConnectionEvent
	for range m.monitorServers {
		initChs = append(initChs, make(chan *networkservice.ConnectionEvent, 1))
	}

	errCh := make(chan error, len(initChs))

	var monitorErr atomic.Value
	for monitorServer, i := range m.monitorServers {
		go startMonitorConnectionsServer(ctx, cancel, initChs[i], errCh, selector, rawSrv, monitorServer, &monitorErr)
	}
	processInitEvent(ctx, initChs, errCh, rawSrv)

	<-ctx.Done()

	var err error
	if rv := monitorErr.Load(); rv != nil {
		err = rv.(error)
	}
	return err
}

func startMonitorConnectionsServer(
	ctx context.Context, cancel context.CancelFunc,
	initCh chan<- *networkservice.ConnectionEvent, errCh <-chan error,
	selector *networkservice.MonitorScopeSelector, rawSrv networkservice.MonitorConnection_MonitorConnectionsServer,
	monitorServer networkservice.MonitorConnectionServer,
	monitorErr *atomic.Value,
) {
	srv := &combineMonitorConnectionsServer{
		ctx:    ctx,
		initCh: initCh,
		errCh:  errCh,
		MonitorConnection_MonitorConnectionsServer: rawSrv,
	}
	srv.initWg.Add(1)

	defer func() {
		cancel()
		srv.initOnce.Do(srv.initWg.Done)
	}()

	if err := monitorServer.MonitorConnections(selector, srv); err != nil {
		monitorErr.Store(err)
	}
}

func processInitEvent(
	ctx context.Context,
	initChs []chan *networkservice.ConnectionEvent, errCh chan error,
	rawSrv networkservice.MonitorConnection_MonitorConnectionsServer,
) {
	defer close(errCh)

	initEvent := &networkservice.ConnectionEvent{
		Type:        networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
		Connections: make(map[string]*networkservice.Connection),
	}
	for _, initCh := range initChs {
		select {
		case <-ctx.Done():
			return
		case event := <-initCh:
			for id, conn := range event.Connections {
				initEvent.Connections[id] = conn
			}
		}
	}

	if initErr := rawSrv.Send(initEvent); initErr != nil {
		for i := 0; i < len(initChs); i++ {
			errCh <- initErr
		}
	}
}

type combineMonitorConnectionsServer struct {
	ctx      context.Context
	initCh   chan<- *networkservice.ConnectionEvent
	initOnce sync.Once
	initWg   sync.WaitGroup
	errCh    <-chan error

	networkservice.MonitorConnection_MonitorConnectionsServer
}

func (m *combineMonitorConnectionsServer) Send(event *networkservice.ConnectionEvent) error {
	switch event.Type {
	case networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER:
		err := errors.New("double sending initial state transfer")
		m.initOnce.Do(func() {
			defer m.initWg.Done()

			m.initCh <- event
			err = <-m.errCh
		})
		return err
	default:
		m.initWg.Wait()
		return m.MonitorConnection_MonitorConnectionsServer.Send(event)
	}
}

func (m *combineMonitorConnectionsServer) Context() context.Context {
	return m.ctx
}
