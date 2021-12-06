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

// Package vl3ipam provides a vl3 IPAM server chain element.
package vl3ipam

import (
	"context"
	"net"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type info struct {
	/* IP */
	ipPool *ippool.IPPool
	ip     string

	/* count - the number of clients using this IPs */
	count uint32
}

type serviceNameMap struct {
	/* entries - is a map[NetworkServiceName]{} */
	entries map[string]*info

	/* mutex for entries */
	mut sync.Mutex
}

type ipamServer struct {
	Map
	serviceMap serviceNameMap
	ipPools    []*ippool.IPPool
	prefixes   []*net.IPNet
	own        *net.IPNet
	once       sync.Once
	initErr    error
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
func NewServer(own *net.IPNet, prefixes ...*net.IPNet) networkservice.NetworkServiceServer {
	return &ipamServer{
		own:      own,
		prefixes: prefixes,
		serviceMap: serviceNameMap{
			entries: make(map[string]*info),
		},
	}
}

func (s *ipamServer) init(conn *networkservice.Connection) {
	if len(s.prefixes) == 0 {
		s.initErr = errors.New("required one or more prefixes")
		return
	}

	for _, prefix := range s.prefixes {
		if prefix == nil {
			s.initErr = errors.Errorf("prefix must not be nil: %+v", s.prefixes)
			return
		}
		pool := ippool.NewWithNet(prefix)
		if s.own != nil {
			/*  If own IP is exists - exclude it from all of the pools    */
			/*  because we don't know from which prefix it was allocated  */
			pool.Exclude(s.own)
			if conn != nil {
				s.serviceMap.entries[conn.GetNetworkService()] = &info{
					ipPool: pool,
					ip:     s.own.String(),
					count:  1,
				}
			}
		}
		s.ipPools = append(s.ipPools, pool)
	}
}

func (s *ipamServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	s.once.Do(func() {
		s.init(request.GetConnection())
	})
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
	ipContext.SrcIpRequired = true

	excludeIP4, excludeIP6 := exclude(ipContext.GetExcludedPrefixes()...)

	if err := s.vl3ClientProcess(ctx, ipContext, excludeIP4, excludeIP6); err != nil {
		return nil, err
	}

	nsName := request.GetConnection().GetNetworkService()
	connInfo, loaded := s.Load(conn.GetId())
	var err error
	if loaded && (connInfo.shouldUpdate(excludeIP4) || connInfo.shouldUpdate(excludeIP6)) {
		// some of the existing addresses are excluded
		deleteAddr(&ipContext.SrcIpAddrs, connInfo.srcAddr)
		deleteAddr(&ipContext.DstIpAddrs, connInfo.dstAddr)
		deleteRoute(&ipContext.SrcRoutes, connInfo.dstAddr)
		deleteRoute(&ipContext.DstRoutes, connInfo.srcAddr)
		s.free(connInfo, nsName)
		loaded = false
	}
	if !loaded {
		if connInfo, err = s.getAddr(conn.GetId(), nsName, excludeIP4, excludeIP6, false); err != nil {
			return nil, err
		}
		s.Store(conn.GetId(), connInfo)
		if ipContext.SrcIpRequired {
			if connInfo, err = s.getAddr(conn.GetId(), nsName, excludeIP4, excludeIP6, true); err != nil {
				return nil, err
			}
			ipContext.SrcIpRequired = false
		}
		s.Store(conn.GetId(), connInfo)
	}

	addAddr(&ipContext.SrcIpAddrs, connInfo.srcAddr)
	addRoute(&ipContext.DstRoutes, connInfo.srcAddr)
	addAddr(&ipContext.DstIpAddrs, connInfo.dstAddr)
	for _, pref := range s.prefixes {
		addRoute(&ipContext.SrcRoutes, pref.String())
	}

	conn, err = next.Server(ctx).Request(ctx, request)
	if err != nil {
		if !loaded {
			s.free(connInfo, nsName)
		}
		return nil, err
	}
	return conn, nil
}

//  Allocate IP address for vl3-client.
//  If there are ExtraPrefixes and there are not SrcIpAddrs - allocate new ones
func (s *ipamServer) vl3ClientProcess(ctx context.Context, ipContext *networkservice.IPContext, excludeIP4, excludeIP6 *ippool.IPPool) error {
	if len(ipContext.ExtraPrefixes) == 0 {
		return nil
	}
	if len(ipContext.SrcIpAddrs) == 0 {
		allocated := false
		for _, prefix := range ipContext.ExtraPrefixes {
			_, ipNet, err := net.ParseCIDR(prefix)
			if err != nil {
				log.FromContext(ctx).Errorf("error cidr %v parsing: %v", prefix, err.Error())
				continue
			}

			addr, err := getAddrFromIPPool(ippool.NewWithNet(ipNet), excludeIP4, excludeIP6)
			if err != nil {
				log.FromContext(ctx).Errorf("error getting ip from prefix %v: %v", prefix, err.Error())
				continue
			}
			addAddr(&ipContext.SrcIpAddrs, addr)
			addRoute(&ipContext.DstRoutes, addr)
			allocated = true
			break
		}
		if !allocated {
			return errors.New("error while allocating an address")
		}
	}
	ipContext.SrcIpRequired = false

	return nil
}

func (s *ipamServer) getAddr(connID, nsName string, excludeIP4, excludeIP6 *ippool.IPPool, isSrcIP bool) (connInfo *connectionInfo, err error) {
	s.serviceMap.mut.Lock()
	defer s.serviceMap.mut.Unlock()

	if !isSrcIP {
		// Check if we have already allocated address for the required service
		inf, ok := s.serviceMap.entries[nsName]
		if ok {
			connInfo, loaded := s.Load(connID)
			if !loaded {
				connInfo = &connectionInfo{
					ipPool:  inf.ipPool,
					dstAddr: inf.ip,
				}
				inf.count++
			}
			return connInfo, nil
		}
	}

	var addr net.IP
	for _, ipPool := range s.ipPools {
		for {
			if addr, err = ipPool.Pull(); err != nil {
				break
			}
			if !excludeIP4.ContainsString(addr.String()) && !excludeIP6.ContainsString(addr.String()) {
				connInfo, loaded := s.Load(connID)
				if !loaded {
					connInfo = &connectionInfo{
						ipPool: ipPool,
					}
				}

				ipNet := &net.IPNet{
					IP:   addr,
					Mask: net.CIDRMask(len(addr)*8, len(addr)*8),
				}
				if isSrcIP {
					connInfo.srcAddr = ipNet.String()
				} else {
					connInfo.dstAddr = ipNet.String()
					s.serviceMap.entries[nsName] = &info{
						ip:     connInfo.dstAddr,
						ipPool: connInfo.ipPool,
						count:  1,
					}
				}
				return connInfo, nil
			}
		}
	}
	return nil, err
}

func getAddrFromIPPool(ipPool, excludeIP4, excludeIP6 *ippool.IPPool) (ip string, err error) {
	var addr net.IP
	for {
		if addr, err = ipPool.Pull(); err != nil {
			return "", err
		}
		if !excludeIP4.ContainsString(addr.String()) && !excludeIP6.ContainsString(addr.String()) {
			ipNet := &net.IPNet{
				IP:   addr,
				Mask: net.CIDRMask(len(addr)*8, len(addr)*8),
			}
			return ipNet.String(), nil
		}
	}
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
	if prefix != "" {
		*routes = append(*routes, &networkservice.Route{
			Prefix: prefix,
		})
	}
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
	if addr != "" {
		*addrs = append(*addrs, addr)
	}
}

func (s *ipamServer) Close(ctx context.Context, conn *networkservice.Connection) (_ *empty.Empty, err error) {
	if s.initErr != nil {
		return nil, s.initErr
	}

	nsName := conn.GetNetworkService()
	if connInfo, ok := s.Load(conn.GetId()); ok {
		s.free(connInfo, nsName)
	}
	return next.Server(ctx).Close(ctx, conn)
}

func (s *ipamServer) free(connInfo *connectionInfo, nsName string) {
	connInfo.ipPool.AddNetString(connInfo.srcAddr)

	if connInfo.dstAddr != "" {
		s.serviceMap.mut.Lock()
		if info, ok := s.serviceMap.entries[nsName]; ok {
			info.count--
			if info.count == 0 {
				connInfo.ipPool.AddNetString(connInfo.dstAddr)
				delete(s.serviceMap.entries, nsName)
			}
		}
		s.serviceMap.mut.Unlock()
	}
}
