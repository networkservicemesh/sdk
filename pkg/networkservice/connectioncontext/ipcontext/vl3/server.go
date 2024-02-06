// Copyright (c) 2024 Cisco and/or its affiliates.
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

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/ipam"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/edwarnicke/genericsync"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type vl3Server struct {
	pool      vl3IPAM
	subnetMap genericsync.Map[string, string] // Map connectionId:subnet
}

// NewServer - returns a new vL3 server instance that manages connection.context.ipcontext for vL3 scenario.
//
//	Produces refresh on prefix update.
//	Requires begin and metdata chain elements.
func NewServer(ctx context.Context, prefixCh <-chan *ipam.PrefixResponse) networkservice.NetworkServiceServer {
	var result = new(vl3Server)

	go func() {
		for resp := range prefixCh {
			var prefix = resp.GetPrefix()
			result.pool.reset(ctx, prefix, resp.GetExcludePrefixes())
			log.FromContext(ctx).Infof("NewServer. Extracted prefix: %s", prefix)
		}
	}()

	return result
}

// NewDualstackServer - returns a chain of new vL3 server instance that manages connection.context.ipcontext for vL3 scenario.
func NewDualstackServer(ctx context.Context, prefixChs []chan *ipam.PrefixResponse) networkservice.NetworkServiceServer {
	var servers []networkservice.NetworkServiceServer

	for _, prefixCh := range prefixChs {
		servers = append(servers, NewServer(ctx, prefixCh))
	}

	return chain.NewNetworkServiceServer(servers...)
}

func (v *vl3Server) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if !v.pool.isInitialized() {
		return nil, errors.New("prefix pool is initializing")
	}
	if request.Connection == nil {
		request.Connection = new(networkservice.Connection)
	}
	var conn = request.GetConnection()
	if conn.GetContext() == nil {
		conn.Context = new(networkservice.ConnectionContext)
	}
	if conn.GetContext().GetIpContext() == nil {
		conn.GetContext().IpContext = new(networkservice.IPContext)
	}

	var ipContext = conn.GetContext().GetIpContext()

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
	} else {
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
	for _, srcAddr := range ipContext.SrcIpAddrs {
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
		if route.Prefix == prefix {
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
	for _, ip := range ipContext.SrcIpAddrs {
		if !prevIPPool.ContainsNetString(ip) {
			srcIPAddrs = append(srcIPAddrs, ip)
		}
	}
	ipContext.SrcIpAddrs = srcIPAddrs

	var dstIPAddrs []string
	for _, ip := range ipContext.DstIpAddrs {
		if !prevIPPool.ContainsNetString(ip) {
			dstIPAddrs = append(dstIPAddrs, ip)
		}
	}
	ipContext.DstIpAddrs = dstIPAddrs

	var srcRoutes []*networkservice.Route
	for _, r := range ipContext.SrcRoutes {
		if !prevIPPool.ContainsNetString(r.Prefix) {
			srcRoutes = append(srcRoutes, r)
		}
	}
	ipContext.SrcRoutes = srcRoutes

	var dstRoutes []*networkservice.Route
	for _, r := range ipContext.DstRoutes {
		if !prevIPPool.ContainsNetString(r.Prefix) {
			dstRoutes = append(dstRoutes, r)
		}
	}
	ipContext.DstRoutes = dstRoutes
}
