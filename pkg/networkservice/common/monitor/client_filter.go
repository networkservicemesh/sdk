// Copyright (c) 2021 Cisco and/or its affiliates.
//
// Copyright (c) 2023-2024 Cisco Systems, Inc.
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

package monitor

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
)

type clientFilter struct {
	conn *networkservice.Connection
	networkservice.MonitorConnection_MonitorConnectionsClient
}

func newClientFilter(client networkservice.MonitorConnection_MonitorConnectionsClient, conn *networkservice.Connection) networkservice.MonitorConnection_MonitorConnectionsClient {
	return &clientFilter{
		MonitorConnection_MonitorConnectionsClient: client,
		conn: conn,
	}
}

func (c *clientFilter) Recv() (*networkservice.ConnectionEvent, error) {
	for {
		if c == nil || c.MonitorConnection_MonitorConnectionsClient == nil {
			return nil, errors.New("MonitorConnections cilent is nil")
		}
		eventIn, err := c.MonitorConnection_MonitorConnectionsClient.Recv()
		if err != nil {
			return nil, errors.Wrap(err, "MonitorConnections client failed to receive an event")
		}
		eventOut := &networkservice.ConnectionEvent{
			Type:        networkservice.ConnectionEventType_UPDATE,
			Connections: make(map[string]*networkservice.Connection),
		}
		for _, connIn := range eventIn.GetConnections() {
			// If we don't have enough PathSegments connIn doesn't match e.conn
			if len(connIn.GetPath().GetPathSegments()) < int(c.conn.GetPath().GetIndex()+1) {
				continue
			}
			// If the e.conn isn't in the expected PathSegment connIn doesn't match e.conn
			if connIn.GetPath().GetPathSegments()[int(c.conn.GetPath().GetIndex())].GetId() != c.conn.GetId() {
				continue
			}
			// If the current index isn't the index of e.conn or what comes after it connIn doesn't match e.conn
			if !(connIn.GetPath().GetIndex() == c.conn.GetPath().GetIndex() || connIn.GetPath().GetIndex() == c.conn.GetPath().GetIndex()+1) {
				continue
			}

			if eventIn.GetType() == networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER &&
				connIn.GetState() == c.conn.GetState() {
				continue
			}

			// Construct the outgoing Connection
			connOut := c.conn.Clone()
			connOut.Path = connIn.GetPath()
			connOut.GetPath().Index = c.conn.GetPath().GetIndex()
			connOut.Context = connIn.GetContext()
			connOut.State = connIn.GetState()

			// If it's deleted, mark the event state down
			if eventIn.GetType() == networkservice.ConnectionEventType_DELETE {
				connOut.State = networkservice.State_DOWN
			}

			// If the connection hasn't changed... don't send the event
			if connOut.Equals(c.conn) {
				continue
			}

			// Add the Connection to the outgoing event
			eventOut.GetConnections()[connOut.GetId()] = connOut

			// Update the event we are watching for:
			c.conn = connOut
		}
		if len(eventOut.GetConnections()) > 0 {
			return eventOut, nil
		}
	}
}
