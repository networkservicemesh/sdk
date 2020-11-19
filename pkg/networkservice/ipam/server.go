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

// Package ipam provides a simple ipam appropriate for:
// * point 2 point - assigns two ip addresses out of a pool of prefixes with a CIDR mask equal /31
// * IP net - assigns ip address out of a pool of prefixes with a CIDR mask of selected prefix
// The IP addresses assigned are not reassigned until they are released. All subsequent requests return previously
// assigned IP addresses except than the request type has been changed or existing IP addresses are excluded with new
// exclude prefixes.
package ipam

import (
	"context"
	"net"
	"sync"

	"github.com/RoaringBitmap/roaring"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type ipamServer struct {
	ipPools   []*ipPool
	connInfos connectionInfoMap
	prefixes  []*net.IPNet
	once      sync.Once
	initErr   error
}

type connectionInfo struct {
	ipPool  *ipPool
	srcAddr string
	dstAddr string
}

// NewServer - creates a NetworkServiceServer that requests a kernel interface and populates the netns inode
func NewServer(prefixes ...*net.IPNet) networkservice.NetworkServiceServer {
	return &ipamServer{
		prefixes: prefixes,
	}
}

func (s *ipamServer) init() {
	if len(s.prefixes) == 0 {
		s.initErr = errors.New("required one or more prefixes")
		return
	}

	for _, prefix := range s.prefixes {
		if prefix == nil {
			s.initErr = errors.Errorf("prefix must not be nil: %+v", s.prefixes)
			return
		}
		s.ipPools = append(s.ipPools, newIPPool(prefix))
	}
}

func (s *ipamServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	s.once.Do(s.init)
	if s.initErr != nil {
		return nil, s.initErr
	}

	conn := request.GetConnection()
	if conn.GetContext() == nil {
		conn.Context = &networkservice.ConnectionContext{}
	}
	if conn.GetContext().GetIpContext() == nil {
		conn.GetContext().IpContext = &networkservice.IPContext{}
	}
	ipContext := conn.GetContext().GetIpContext()

	exclude, err := exclude(ipContext.GetExcludedPrefixes()...)
	if err != nil {
		return nil, err
	}

	connInfo, ok := s.connInfos.Load(conn.GetId())
	if ok && shouldUpdate(connInfo, ipContext, exclude) {
		// we are requested for another { p2p, IP net } case or some of the existing addresses are excluded
		s.delete(conn.GetId(), connInfo)
		ok = false
	}
	if !ok {
		var srcAddr, dstAddr string
		for _, ipPool := range s.ipPools {
			if ipContext.GetDstIpRequired() {
				// p2p ipam case
				if dstAddr, srcAddr, err = ipPool.getP2PAddrs(exclude); err != nil {
					continue
				}
			} else {
				// IP net ipam case
				if srcAddr, err = ipPool.getIPNetAddr(exclude); err != nil {
					continue
				}
			}
			connInfo = &connectionInfo{
				ipPool:  ipPool,
				srcAddr: srcAddr,
				dstAddr: dstAddr,
			}
			s.connInfos.Store(conn.GetId(), connInfo)
		}
		if err != nil {
			return nil, err
		}
	}

	ipContext.SrcIpAddr = connInfo.srcAddr
	if ipContext.GetDstIpRequired() {
		ipContext.DstIpAddr = connInfo.dstAddr
	}

	return next.Server(ctx).Request(ctx, request)
}

func shouldUpdate(connInfo *connectionInfo, ipContext *networkservice.IPContext, exclude *roaring.Bitmap) bool {
	return ((connInfo.dstAddr != "") != (ipContext.GetDstIpRequired())) ||
		exclude.Contains(addrToInt(connInfo.srcAddr)) ||
		connInfo.dstAddr != "" && exclude.Contains(addrToInt(connInfo.srcAddr))
}

func (s *ipamServer) Close(ctx context.Context, conn *networkservice.Connection) (_ *empty.Empty, err error) {
	s.once.Do(s.init)
	if s.initErr != nil {
		return nil, s.initErr
	}

	if connInfo, ok := s.connInfos.Load(conn.GetId()); ok {
		s.delete(conn.GetId(), connInfo)
	}

	return next.Server(ctx).Close(ctx, conn)
}

func (s *ipamServer) delete(connID string, connInfo *connectionInfo) {
	connInfo.ipPool.freeAddrs(connInfo.srcAddr)
	if dstAddr := connInfo.dstAddr; dstAddr != "" {
		connInfo.ipPool.freeAddrs(dstAddr)
	}
	s.connInfos.Delete(connID)
}
