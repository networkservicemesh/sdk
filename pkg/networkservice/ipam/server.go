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

// Package ipam provides an IPAM server chain element.
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
	ipPools  []*ipPool
	prefixes []*net.IPNet
	once     sync.Once
	initErr  error
}

type connectionInfo struct {
	ipPool  *ipPool
	srcAddr string
	dstAddr string
}

func (i *connectionInfo) shouldUpdate(ipContext *networkservice.IPContext, exclude *roaring.Bitmap) bool {
	return ((i.dstAddr != "") != (ipContext.GetDstIpRequired())) ||
		exclude.Contains(addrToInt(i.srcAddr)) ||
		i.dstAddr != "" && exclude.Contains(addrToInt(i.srcAddr))
}

// NewServer - creates a new NetworkServiceServer chain element that implements IPAM service.
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

	connInfo, ok := loadConnInfo(ctx)
	if ok && connInfo.shouldUpdate(ipContext, exclude) {
		// we are requested for another { p2p, IP subnet } case or some of the existing addresses are excluded
		s.free(connInfo)
		ok = false
	}
	if !ok {
		if ipContext.GetDstIpRequired() {
			// p2p ipam case
			connInfo, err = s.getP2PAddrs(exclude)
		} else {
			// IP net ipam case
			connInfo, err = s.getIPSubnetAddr(exclude)
		}
		if err != nil {
			return nil, err
		}
		storeConnInfo(ctx, connInfo)
	}

	ipContext.SrcIpAddr = connInfo.srcAddr
	if ipContext.GetDstIpRequired() {
		ipContext.DstIpAddr = connInfo.dstAddr
	}

	return next.Server(ctx).Request(ctx, request)
}

func (s *ipamServer) getP2PAddrs(exclude *roaring.Bitmap) (connInfo *connectionInfo, err error) {
	var dstAddr, srcAddr string
	for _, ipPool := range s.ipPools {
		if dstAddr, srcAddr, err = ipPool.getP2PAddrs(exclude); err == nil {
			return &connectionInfo{
				ipPool:  ipPool,
				srcAddr: srcAddr,
				dstAddr: dstAddr,
			}, nil
		}
	}
	return nil, err
}

func (s *ipamServer) getIPSubnetAddr(exclude *roaring.Bitmap) (connInfo *connectionInfo, err error) {
	var srcAddr string
	for _, ipPool := range s.ipPools {
		if srcAddr, err = ipPool.getIPSubnetAddr(exclude); err == nil {
			return &connectionInfo{
				ipPool:  ipPool,
				srcAddr: srcAddr,
			}, nil
		}
	}
	return nil, err
}

func (s *ipamServer) Close(ctx context.Context, conn *networkservice.Connection) (_ *empty.Empty, err error) {
	s.once.Do(s.init)
	if s.initErr != nil {
		return nil, s.initErr
	}

	if connInfo, ok := loadConnInfo(ctx); ok {
		s.free(connInfo)
	}

	return next.Server(ctx).Close(ctx, conn)
}

func (s *ipamServer) free(connInfo *connectionInfo) {
	connInfo.ipPool.freeAddrs(connInfo.srcAddr)
	if dstAddr := connInfo.dstAddr; dstAddr != "" {
		connInfo.ipPool.freeAddrs(dstAddr)
	}
}
