// Copyright (c) 2020 Cisco and/or its affiliates.
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

// Package peertracker provides a wrapper for a Nsmgr that tracks connections received from local Clients
// Its designed to be used in a DevicePlugin to allow us to properly Close connections on re-Allocate
package peertracker

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc/peer"

	"github.com/edwarnicke/serialize"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
)

const (
	unixScheme = "unix"
)

type peerTrackerServer struct {
	nsmgr.Nsmgr
	executor serialize.Executor
	// Outer map is peer url.URL.String(), inner map key is Connection.Id
	connections map[string]map[string]*networkservice.Connection
}

// NewServer - Creates a new peer tracker Server
//           inner - Nsmgr being wrapped
//           closeAll - pointer to memory location to which you should write a pointer to the function to be called
//                      to close all connections for the provided url (presuming a unix URL)
func NewServer(inner nsmgr.Nsmgr, closeAll *func(ctx context.Context, u *url.URL)) nsmgr.Nsmgr {
	rv := &peerTrackerServer{
		connections: make(map[string]map[string]*networkservice.Connection),
		Nsmgr:       inner,
	}
	*closeAll = rv.closeAllConnectionsForPeer
	return rv
}

func (p *peerTrackerServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := p.Nsmgr.Request(ctx, request)
	if err != nil {
		return nil, err
	}
	mypeer, ok := peer.FromContext(ctx)
	if ok {
		if mypeer.Addr.Network() == unixScheme {
			u := &url.URL{
				Scheme: mypeer.Addr.Network(),
				Path:   mypeer.Addr.String(),
			}
			p.executor.AsyncExec(func() {
				_, ok := p.connections[u.String()]
				if !ok {
					p.connections[u.String()] = make(map[string]*networkservice.Connection)
				}
				p.connections[u.String()][conn.GetId()] = conn
			})
		}
	}
	return conn, nil
}

func (p *peerTrackerServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, err := p.Nsmgr.Close(ctx, conn)
	if err != nil {
		return nil, err
	}
	mypeer, ok := peer.FromContext(ctx)
	if ok {
		if mypeer.Addr.Network() == unixScheme {
			u := &url.URL{
				Scheme: mypeer.Addr.Network(),
				Path:   mypeer.Addr.String(),
			}
			p.executor.AsyncExec(func() {
				delete(p.connections[u.String()], conn.GetId())
			})
		}
	}
	return &empty.Empty{}, nil
}

func (p *peerTrackerServer) closeAllConnectionsForPeer(ctx context.Context, u *url.URL) {
	finishedChan := make(chan struct{})
	<-p.executor.AsyncExec(func() {
		if connMap, ok := p.connections[u.String()]; ok {
			for _, conn := range connMap {
				_, _ = p.Close(ctx, conn)
			}
		}
		close(finishedChan)
	})
}
