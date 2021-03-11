// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

// Package point2pointipam provides a p2p IPAM server chain element.
package point2pointipam

import (
	"context"
	"net"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
)

type ipamServer struct {
	ipPools  []*ippool.IPPool
	prefixes []*net.IPNet
	once     sync.Once
	initErr  error
}

type connectionInfo struct {
	ipPool  *ippool.IPPool
	srcAddr string
	dstAddr string
}

func (i *connectionInfo) shouldUpdate(exclude *ippool.IPPool) bool {
	srcIP, _, srcErr := net.ParseCIDR(i.srcAddr)
	dstIP, _, dstErr := net.ParseCIDR(i.dstAddr)

	return srcErr == nil && exclude.ContainsString(srcIP.String()) || dstErr == nil && exclude.ContainsString(dstIP.String())
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
		s.ipPools = append(s.ipPools, ippool.NewWithNet(prefix))
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

	excludeIP4, excludeIP6 := exclude(ipContext.GetExcludedPrefixes()...)

	connInfo, ok := loadConnInfo(ctx)
	var err error
	if ok && (connInfo.shouldUpdate(excludeIP4) || connInfo.shouldUpdate(excludeIP6)) {
		// some of the existing addresses are excluded
		deleteRoute(&ipContext.SrcRoutes, connInfo.dstAddr)
		deleteRoute(&ipContext.DstRoutes, connInfo.srcAddr)
		s.free(connInfo)
		ok = false
	}
	if !ok {
		if connInfo, err = s.getP2PAddrs(excludeIP4, excludeIP6); err != nil {
			return nil, err
		}
		storeConnInfo(ctx, connInfo)
	}

	ipContext.SrcIpAddr = connInfo.srcAddr
	addRoute(&ipContext.SrcRoutes, connInfo.dstAddr)

	ipContext.DstIpAddr = connInfo.dstAddr
	addRoute(&ipContext.DstRoutes, connInfo.srcAddr)

	return next.Server(ctx).Request(ctx, request)
}

func (s *ipamServer) getP2PAddrs(excludeIP4, excludeIP6 *ippool.IPPool) (connInfo *connectionInfo, err error) {
	var dstAddr, srcAddr *net.IPNet
	for _, ipPool := range s.ipPools {
		if dstAddr, srcAddr, err = ipPool.PullP2PAddrs(excludeIP4, excludeIP6); err == nil {
			return &connectionInfo{
				ipPool:  ipPool,
				srcAddr: srcAddr.String(),
				dstAddr: dstAddr.String(),
			}, nil
		}
	}
	return nil, err
}

func deleteRoute(routes *[]*networkservice.Route, prefix string) {
	for i, route := range *routes {
		if route.Prefix == prefix {
			*routes = append((*routes)[:i], (*routes)[i+1:]...)
			return
		}
	}
}

func addRoute(routes *[]*networkservice.Route, prefix string) {
	for _, route := range *routes {
		if route.Prefix == prefix {
			return
		}
	}
	*routes = append(*routes, &networkservice.Route{
		Prefix: prefix,
	})
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
	connInfo.ipPool.AddNetString(connInfo.srcAddr)
	connInfo.ipPool.AddNetString(connInfo.dstAddr)
}
