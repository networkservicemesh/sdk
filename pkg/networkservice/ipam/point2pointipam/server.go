// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type ipamServer struct {
	genericsync.Map[string, *connectionInfo]
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
		return nil, errors.Wrap(s.initErr, "failed to init IPAM server during request")
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

	connInfo, loaded := s.Load(conn.GetId())
	var err error
	if loaded && (connInfo.shouldUpdate(excludeIP4) || connInfo.shouldUpdate(excludeIP6)) {
		// some of the existing addresses are excluded
		deleteAddr(&ipContext.SrcIpAddrs, connInfo.srcAddr)
		deleteAddr(&ipContext.DstIpAddrs, connInfo.dstAddr)
		deleteRoute(&ipContext.SrcRoutes, connInfo.dstAddr)
		deleteRoute(&ipContext.DstRoutes, connInfo.srcAddr)
		s.free(connInfo)
		loaded = false
	}
	// If not loaded, we are primarily trying to recover addresses from the IpContext. In case of an error, allocate new ones.
	if !loaded {
		if connInfo, err = s.recoverAddrs(ipContext.GetSrcIpAddrs(), ipContext.GetDstIpAddrs(), excludeIP4, excludeIP6); err == nil {
			log.FromContext(ctx).Infof("addresses have been recovered - srcIP: %v, dstIP: %v", connInfo.srcAddr, connInfo.dstAddr)
		} else if connInfo, err = s.getP2PAddrs(excludeIP4, excludeIP6); err != nil {
			return nil, err
		}
		s.Store(conn.GetId(), connInfo)
	}

	ipContext = &networkservice.IPContext{}

	addAddr(&ipContext.SrcIpAddrs, connInfo.srcAddr)
	addRoute(&ipContext.SrcRoutes, connInfo.dstAddr)

	addAddr(&ipContext.DstIpAddrs, connInfo.dstAddr)
	addRoute(&ipContext.DstRoutes, connInfo.srcAddr)

	conn, err = next.Server(ctx).Request(ctx, request)
	if err != nil {
		if !loaded {
			s.free(connInfo)
		}
		return nil, err
	}

	return conn, nil
}

func (s *ipamServer) recoverAddrs(srcAddrs, dstAddrs []string, excludeIP4, excludeIP6 *ippool.IPPool) (connInfo *connectionInfo, err error) {
	if len(srcAddrs) == 0 || len(dstAddrs) == 0 {
		return nil, errors.New("addresses cannot be empty for recovery")
	}
	for _, ipPool := range s.ipPools {
		var srcAddr, dstAddr *net.IPNet
		for i, addr := range srcAddrs {

			if srcAddr, err = ipPool.PullIPString(addr, excludeIP4, excludeIP6); err == nil {
				break
			} else {
				srcAddrs = append(srcAddrs[:i], srcAddrs[i+1:]...)
			}
		}
		for i, addr := range dstAddrs {
			if dstAddr, err = ipPool.PullIPString(addr, excludeIP4, excludeIP6); err == nil {
				break
			} else {
				dstAddrs = append(dstAddrs[:i], dstAddrs[i+1:]...)
			}
		}
		if srcAddr != nil && dstAddr != nil {
			return &connectionInfo{
				ipPool:  ipPool,
				srcAddr: srcAddr.String(),
				dstAddr: dstAddr.String(),
			}, nil
		}
	}

	return nil, errors.Errorf("unable to recover: %+v, %+v", srcAddrs, dstAddrs)
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

func deleteAddr(addrs *[]string, addr string) {
	for i, a := range *addrs {
		if a == addr {
			*addrs = append((*addrs)[:i], (*addrs)[i+1:]...)
			return
		}
	}
}

func addAddr(addrs *[]string, addr string) {
	for _, a := range *addrs {
		if a == addr {
			return
		}
	}
	*addrs = append(*addrs, addr)
}

func (s *ipamServer) Close(ctx context.Context, conn *networkservice.Connection) (_ *empty.Empty, err error) {
	s.once.Do(s.init)
	if s.initErr != nil {
		return nil, errors.Wrap(s.initErr, "failed to init IPAM server during close")
	}

	if connInfo, ok := s.LoadAndDelete(conn.GetId()); ok {
		s.free(connInfo)
	}

	return next.Server(ctx).Close(ctx, conn)
}

func (s *ipamServer) free(connInfo *connectionInfo) {
	connInfo.ipPool.AddNetString(connInfo.srcAddr)
	connInfo.ipPool.AddNetString(connInfo.dstAddr)
}
