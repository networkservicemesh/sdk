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

// Package monitor provides a TestMonitorClient test class to perform client monitoring based on MonitorConnection call.
package monitor

import (
	"context"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"time"
)

// TestMonitorClient - implementation of monitor client.
type TestMonitorClient struct {
	Events       []*networkservice.ConnectionEvent
	eventChannel chan *networkservice.ConnectionEvent
	ctx          context.Context
	Cancel       context.CancelFunc
	grpc.ServerStream
}

// NewTestMonitorClient - construct a new client.
func NewTestMonitorClient() *TestMonitorClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &TestMonitorClient{
		eventChannel: make(chan *networkservice.ConnectionEvent, 10),
		ctx:          ctx,
		Cancel:       cancel,
	}
}

// Send - recieve event from server.
func (t *TestMonitorClient) Send(evt *networkservice.ConnectionEvent) error {
	t.Events = append(t.Events, evt)
	t.eventChannel <- evt
	return nil
}

// Context - current context to perform checks.
func (t *TestMonitorClient) Context() context.Context {
	return t.ctx
}

// BeginMonitoring - start monitoring inside go routine, use Cancel() to perform stop.
func (t *TestMonitorClient) BeginMonitoring(server networkservice.MonitorConnectionServer, segmentName string) {
	go func() {
		_ = server.MonitorConnections(
			&networkservice.MonitorScopeSelector{
				PathSegments: []*networkservice.PathSegment{{Name: segmentName}},
			}, t)
	}()
}

// WaitEvents - wait for a required number of events to be recieved.
func (t *TestMonitorClient) WaitEvents(ctx context.Context, count int) {
	for {
		if len(t.Events) == count {
			logrus.Infof("Waiting for Events %v, but has %v", count, len(t.Events))
			break
		}
		// Wait 10ms for listeners to activate
		select {
		case <-ctx.Done():
			// Context is done, we need to exit
			logrus.Errorf("Failed to wait for Events count %v current value is: %v", count, len(t.Events))
			return
		case <-t.eventChannel:
		case <-time.After(1 * time.Second):
		}
	}
}
