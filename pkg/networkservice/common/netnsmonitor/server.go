// Copyright (c) 2020 Cisco Systems, Inc.
//
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

package netnsmonitor

import (
	"context"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/logger"
)

const (
	checkConnectionIntervalDefault = 100 * time.Millisecond
)

type netNSMonitorServer struct {
	ctx         context.Context
	connections connectionMap
}

// NewServer returns new NetNSMonitorServer chain item
func NewServer(ctx context.Context) networkservice.NetworkServiceServer {
	return NewServerWithPeriod(ctx, checkConnectionIntervalDefault)
}

// NewServerWithPeriod returns new NetNSMonitorServer chain item
func NewServerWithPeriod(ctx context.Context, period time.Duration) networkservice.NetworkServiceServer {
	rv := &netNSMonitorServer{
		ctx: ctx,
	}

	go rv.monitorNetNSInode(period)
	return rv
}

func (m *netNSMonitorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)

	if conn.GetMechanism().GetType() == kernel.MECHANISM {
		m.connections.Store(conn.GetId(), conn.Clone())
	}

	return conn, err
}

func (m *netNSMonitorServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	m.connections.Delete(conn.GetId())
	return next.Server(ctx).Close(ctx, conn)
}

func (m *netNSMonitorServer) monitorNetNSInode(period time.Duration) {
	for {
		<-time.After(period)
		m.checkConnectionLiveness()
	}
}

func (m *netNSMonitorServer) checkConnectionLiveness() {
	logEntry := logger.Log(m.ctx).WithField("netNSMonitorServer", "checkConnectionLiveness")

	inodes, err := GetAllNetNS()
	if err != nil {
		logEntry.Errorf("failed to get NetNS: %+v", err)
		return
	}

	inodeSet := NewInodeSet(inodes)
	m.connections.Range(func(id string, conn *networkservice.Connection) bool {
		inode, err := strconv.ParseUint(kernel.ToMechanism(conn.GetMechanism()).GetNetNSInode(), 10, 64)
		if err != nil {
			logEntry.Errorf("failed to convert netNSInode to uint64 number")
			return true
		}

		if !inodeSet.Contains(inode) && conn.GetState() == networkservice.State_UP {
			logEntry.Infof("connection is down")
			m.connections.Delete(conn.GetId())
			if _, err = next.Server(m.ctx).Close(m.ctx, conn); err != nil {
				logEntry.Errorf("failed to close connection: %v %+v", conn.GetId(), err)
			}
		}

		return true
	})
}
