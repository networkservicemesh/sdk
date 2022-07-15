// Copyright (c) 2020-2022 Cisco Systems, Inc.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

// Package monitor provides a NetworkServiceServer chain element to provide a monitor server that reflects
// the connections actually in the NetworkServiceServer
package monitor

import (
	"context"
	"crypto/x509"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	authMonitor "github.com/networkservicemesh/sdk/pkg/tools/monitor/authorize"
	nextMonitor "github.com/networkservicemesh/sdk/pkg/tools/monitor/next"
)

type monitorServer struct {
	chainCtx              context.Context
	spiffeIDConnectionMap *authMonitor.SpiffeIDConnectionMap
	filters               map[string]*monitorFilter
	executor              *serialize.Executor
	connections           map[string]*networkservice.Connection
	networkservice.MonitorConnectionServer
}

// NewServer - creates a NetworkServiceServer chain element that will properly update a MonitorConnectionServer
//             - monitorServerPtr - *networkservice.MonitorConnectionServer.  Since networkservice.MonitorConnectionServer is an interface
//                        (and thus a pointer) *networkservice.MonitorConnectionServer is a double pointer.  Meaning it
//                        points to a place that points to a place that implements networkservice.MonitorConnectionServer
//                        This is done so that we can preserve the return of networkservice.NetworkServer and use
//                        NewServer(...) as any other chain element constructor, but also get back a
//                        networkservice.MonitorConnectionServer that can be used either standalone or in a
//                        networkservice.MonitorConnectionServer chain
//             chainCtx - context for lifecycle management
func NewServer(chainCtx context.Context, monitorServerPtr *networkservice.MonitorConnectionServer) networkservice.NetworkServiceServer {
	spiffeIDConnectionMap := authMonitor.SpiffeIDConnectionMap{}
	filters := make(map[string]*monitorFilter)
	executor := serialize.Executor{}
	connections := make(map[string]*networkservice.Connection)

	*monitorServerPtr = nextMonitor.NewMonitorConnectionServer(
		authMonitor.NewMonitorConnectionServer(&spiffeIDConnectionMap),
		newMonitorConnectionServer(chainCtx, &executor, filters, connections),
	)
	return &monitorServer{
		chainCtx:                chainCtx,
		MonitorConnectionServer: *monitorServerPtr,
		spiffeIDConnectionMap:   &spiffeIDConnectionMap,
		filters:                 filters,
		executor:                &executor,
		connections:             connections,
	}
}

func (m *monitorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	closeCtxFunc := postpone.ContextWithValues(ctx)
	// Cancel any existing eventLoop
	if cancelEventLoop, loaded := loadAndDelete(ctx, metadata.IsClient(m)); loaded {
		cancelEventLoop()
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	spiffeID, err := getSpiffeID(ctx)
	if err == nil {
		ids, _ := m.spiffeIDConnectionMap.Load(spiffeID)
		m.spiffeIDConnectionMap.LoadOrStore(spiffeID, append(ids, conn.GetId()))
	}
	_ = m.Send(&networkservice.ConnectionEvent{
		Type:        networkservice.ConnectionEventType_UPDATE,
		Connections: map[string]*networkservice.Connection{conn.GetId(): conn.Clone()},
	})

	// If we have a clientconn ... we must be part of a passthrough server, and have a client to pass
	// events through from, so start an eventLoop
	cc, ccLoaded := clientconn.Load(ctx)
	if ccLoaded {
		cancelEventLoop, eventLoopErr := newEventLoop(m.chainCtx, m, cc, conn)
		if eventLoopErr != nil {
			closeCtx, closeCancel := closeCtxFunc()
			defer closeCancel()
			_, _ = next.Client(closeCtx).Close(closeCtx, conn)
			return nil, errors.Wrap(eventLoopErr, "unable to monitor")
		}
		store(ctx, metadata.IsClient(m), cancelEventLoop)
	}

	return conn, nil
}

func (m *monitorServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// Cancel any existing eventLoop
	if cancelEventLoop, loaded := loadAndDelete(ctx, metadata.IsClient(m)); loaded {
		cancelEventLoop()
	}
	spiffeID, err := getSpiffeID(ctx)
	if err == nil {
		ids, _ := m.spiffeIDConnectionMap.Load(spiffeID)
		m.spiffeIDConnectionMap.LoadOrStore(spiffeID, append(ids, conn.GetId()))
	}

	rv, err := next.Server(ctx).Close(ctx, conn)
	_ = m.Send(&networkservice.ConnectionEvent{
		Type:        networkservice.ConnectionEventType_DELETE,
		Connections: map[string]*networkservice.Connection{conn.GetId(): conn.Clone()},
	})
	return rv, err
}

func (m *monitorServer) Send(event *networkservice.ConnectionEvent) (_ error) {
	m.executor.AsyncExec(func() {
		if event.Type == networkservice.ConnectionEventType_UPDATE {
			for _, conn := range event.GetConnections() {
				m.connections[conn.GetId()] = conn.Clone()
			}
		}
		if event.Type == networkservice.ConnectionEventType_DELETE {
			for _, conn := range event.GetConnections() {
				delete(m.connections, conn.GetId())
			}
		}
		if event.Type == networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER {
			// sending event with INIITIAL_STATE_TRANSFER not permitted
			return
		}
		for id, filter := range m.filters {
			id, filter := id, filter
			e := event.Clone()
			filter.executor.AsyncExec(func() {
				var err error
				select {
				case <-filter.Context().Done():
					m.executor.AsyncExec(func() {
						delete(m.filters, id)
					})
				default:
					err = filter.Send(e)
				}
				if err != nil {
					m.executor.AsyncExec(func() {
						delete(m.filters, id)
					})
				}
			})
		}
	})
	return nil
}

func getSpiffeID(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	var cert *x509.Certificate
	if ok {
		cert = opa.ParseX509Cert(p.AuthInfo)
	}
	spiffeID, err := x509svid.IDFromCert(cert)
	if err == nil {
		return spiffeID.String(), nil
	}
	return "", err
}
