// Copyright (c) 2022-2024 Cisco and/or its affiliates.
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

package vl3

import (
	"context"
	"net"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/edwarnicke/genericsync"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type vl3Server struct {
	pool      *IPAM
	subnetMap genericsync.Map[string, string] // Map connectionId:subnet
}

// NewServer - returns a new vL3 server instance that manages connection.context.ipcontext for vL3 scenario.
//
//	Produces refresh on prefix update.
//	Requires begin and metdata chain elements.
func NewServer(ctx context.Context, pool *IPAM) networkservice.NetworkServiceServer {
	return &vl3Server{pool: pool}
}

func (v *vl3Server) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if !v.pool.isInitialized() {
		return nil, errors.New("prefix pool is initializing")
	}
	if request.GetConnection() == nil {
		request.Connection = new(networkservice.Connection)
	}
	conn := request.GetConnection()
	if conn.GetContext() == nil {
		conn.Context = new(networkservice.ConnectionContext)
	}
	if conn.GetContext().GetIpContext() == nil {
		conn.GetContext().IpContext = new(networkservice.IPContext)
	}

	ipContext := conn.GetContext().GetIpContext()

	if prevAddress, ok := v.subnetMap.Load(conn.GetId()); ok {
		// Remove previous prefix from IP Context if a current server prefix has changed
		if v.pool.globalIPNet().String() != prevAddress {
			srcNet, err := v.pool.allocate()
			log.FromContext(ctx).Infof("Server Request. Allocated net: %+v for connection: %+v", srcNet.String(), conn.GetId())
			if err != nil {
				return nil, err
			}
			removePreviousPrefixFromIPContext(ipContext, prevAddress)
			ipContext.SrcIpAddrs = append(ipContext.SrcIpAddrs, srcNet.String())
			v.subnetMap.Store(conn.GetId(), v.pool.globalIPNet().String())
		}
	} else if len(request.GetConnection().GetContext().GetDnsContext().GetConfigs()) == 0 || countTheSameVersionSrcIps(request, len(v.pool.self.IP)) == 0 { // TODO Consider a better option to determine vL3NSE-to-vL3NSE server case
		srcNet, err := v.pool.allocate()
		log.FromContext(ctx).Infof("Server Request. Allocated initial net: %+v for connection: %+v", srcNet.String(), conn.GetId())
		if err != nil {
			return nil, err
		}
		ipContext.SrcIpAddrs = append(ipContext.SrcIpAddrs, srcNet.String())
		v.subnetMap.Store(conn.GetId(), v.pool.globalIPNet().String())
	}

	addRoute(&ipContext.SrcRoutes, v.pool.selfAddress().String(), v.pool.selfAddress().IP.String())
	addRoute(&ipContext.SrcRoutes, v.pool.selfPrefix().String(), v.pool.selfAddress().IP.String())
	for _, srcAddr := range ipContext.GetSrcIpAddrs() {
		addRoute(&ipContext.DstRoutes, srcAddr, "")
	}
	addAddr(&ipContext.DstIpAddrs, v.pool.selfAddress().String())

	resp, err := next.Server(ctx).Request(ctx, request)
	if err == nil {
		addRoute(&resp.GetContext().GetIpContext().SrcRoutes, v.pool.globalIPNet().String(), v.pool.selfAddress().IP.String())
	}
	return resp, err
}

func (v *vl3Server) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	for _, srcAddr := range conn.GetContext().GetIpContext().GetSrcIpAddrs() {
		v.pool.freeIfAllocated(srcAddr)
	}
	v.subnetMap.Delete(conn.GetId())
	return next.Server(ctx).Close(ctx, conn)
}

func addRoute(routes *[]*networkservice.Route, prefix, nextHop string) {
	for _, route := range *routes {
		if route.GetPrefix() == prefix {
			return
		}
	}
	*routes = append(*routes, &networkservice.Route{
		Prefix:  prefix,
		NextHop: nextHop,
	})
}

func addAddr(addrs *[]string, addr string) {
	for _, a := range *addrs {
		if a == addr {
			return
		}
	}
	*addrs = append(*addrs, addr)
}

func removePreviousPrefixFromIPContext(ipContext *networkservice.IPContext, prevAddress string) {
	prevIPPool := ippool.NewWithNetString(prevAddress)

	var srcIPAddrs []string
	for _, ip := range ipContext.GetSrcIpAddrs() {
		if !prevIPPool.ContainsNetString(ip) {
			srcIPAddrs = append(srcIPAddrs, ip)
		}
	}
	ipContext.SrcIpAddrs = srcIPAddrs

	var dstIPAddrs []string
	for _, ip := range ipContext.GetDstIpAddrs() {
		if !prevIPPool.ContainsNetString(ip) {
			dstIPAddrs = append(dstIPAddrs, ip)
		}
	}
	ipContext.DstIpAddrs = dstIPAddrs

	var srcRoutes []*networkservice.Route
	for _, r := range ipContext.GetSrcRoutes() {
		if !prevIPPool.ContainsNetString(r.GetPrefix()) {
			srcRoutes = append(srcRoutes, r)
		}
	}
	ipContext.SrcRoutes = srcRoutes

	var dstRoutes []*networkservice.Route
	for _, r := range ipContext.GetDstRoutes() {
		if !prevIPPool.ContainsNetString(r.GetPrefix()) {
			dstRoutes = append(dstRoutes, r)
		}
	}
	ipContext.DstRoutes = dstRoutes
}

func countTheSameVersionSrcIps(request *networkservice.NetworkServiceRequest, selfIPLen int) int {
	srcAddrs := request.GetConnection().GetContext().GetIpContext().GetSrcIpAddrs()
	count := 0
	for _, ip := range srcAddrs {
		_, ipNet, err := net.ParseCIDR(ip)
		if err == nil && len(ipNet.IP) == selfIPLen {
			count++
		}
	}
	return count
}
