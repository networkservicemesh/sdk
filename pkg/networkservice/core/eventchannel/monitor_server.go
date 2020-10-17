// Copyright (c) 2020 Cisco and/or its affiliates.
//
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

package eventchannel

import (
	"errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/edwarnicke/serialize"
)

type monitorConnectionServer struct {
	eventCh   <-chan *networkservice.ConnectionEvent
	closeCh   chan struct{}
	servers   []networkservice.MonitorConnection_MonitorConnectionsServer
	selectors []*networkservice.MonitorScopeSelector
	executor  serialize.Executor
	connectCh chan<- int
}

// NewMonitorServer - returns a networkservice.MonitorConnectionServer
//                    eventCh - when Send() is called on any of the NewMonitorConnection_MonitorConnectionsServers
//                              returned by a call to MonitorConnections, it is inserted into eventCh
func NewMonitorServer(eventCh <-chan *networkservice.ConnectionEvent, options ...MonitorConnectionServerOption) networkservice.MonitorConnectionServer {
	rv := &monitorConnectionServer{
		eventCh: eventCh,
		closeCh: make(chan struct{}),
	}
	for _, o := range options {
		o.apply(rv)
	}
	rv.eventLoop()
	return rv
}

func (m *monitorConnectionServer) MonitorConnections(selector *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) error {
	select {
	case <-m.closeCh:
		return errors.New("sending is no longer possible")
	default:
		m.executor.AsyncExec(func() {
			m.servers = append(m.servers, srv)
			m.selectors = append(m.selectors, selector)
			if m.connectCh != nil {
				m.connectCh <- len(m.servers)
			}
		})
		select {
		case <-srv.Context().Done():
		case <-m.closeCh:
		}
		m.executor.AsyncExec(func() {
			var newServers []networkservice.MonitorConnection_MonitorConnectionsServer
			var newSelectors []*networkservice.MonitorScopeSelector
			for i := range m.servers {
				if m.servers[i] != srv {
					newServers = append(newServers, m.servers[i])
					newSelectors = append(newSelectors, m.selectors[i])
				}
			}
			m.servers = newServers
			m.selectors = newSelectors
		})
		return nil
	}
}

func (m *monitorConnectionServer) eventLoop() {
	go func() {
		for event := range m.eventCh {
			e := event
			m.executor.AsyncExec(func() {
				for i, srv := range m.servers {
					filteredEvent := &networkservice.ConnectionEvent{
						Type:        e.Type,
						Connections: networkservice.FilterMapOnManagerScopeSelector(e.GetConnections(), m.selectors[i]),
					}
					if filteredEvent.Type == networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER || len(filteredEvent.GetConnections()) > 0 {
						// TODO - figure out what if any error handling to do here
						_ = srv.Send(filteredEvent)
					}
				}
			})
		}
		m.executor.AsyncExec(func() {
			close(m.closeCh)
		})
	}()
}
