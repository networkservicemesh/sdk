// Copyright (c) 2022 Cisco and/or its affiliates.
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
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/ipam"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type vl3Server struct {
	pool vl3IPAM
}

// NewServer - returns a new vL3 server instance that manages connection.context.ipcontext for vL3 scenario.
//
//	Produces refresh on prefix update.
//	Requires begin and metdata chain elements.
func NewServer(ctx context.Context, prefixCh <-chan *ipam.PrefixResponse) networkservice.NetworkServiceServer {
	var result = new(vl3Server)

	go func() {
		for resp := range prefixCh {
			result.pool.reset(ctx, resp.GetPrefix(), resp.GetExcludePrefixes())
		}
	}()

	return result
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

	var ipContext = &networkservice.IPContext{
		SrcIpAddrs:       request.GetConnection().Context.GetIpContext().GetSrcIpAddrs(),
		DstRoutes:        request.GetConnection().Context.GetIpContext().GetDstRoutes(),
		ExcludedPrefixes: request.GetConnection().Context.GetIpContext().GetExcludedPrefixes(),
	}

	shouldAllocate := len(ipContext.SrcIpAddrs) == 0

	if prevAddress, ok := loadAddress(ctx); ok && !shouldAllocate {
		shouldAllocate = !v.pool.isExcluded(prevAddress)
	}

	if shouldAllocate {
		srcNet, err := v.pool.allocate()
		if err != nil {
			return nil, err
		}
		ipContext.DstRoutes = nil
		ipContext.SrcIpAddrs = append([]string(nil), srcNet.String())
		storeAddress(ctx, srcNet.String())
	}

	addRoute(&ipContext.SrcRoutes, v.pool.selfAddress().String(), v.pool.selfAddress().IP.String())
	addRoute(&ipContext.SrcRoutes, v.pool.selfPrefix().String(), v.pool.selfAddress().IP.String())
	for _, srcAddr := range ipContext.SrcIpAddrs {
		addRoute(&ipContext.DstRoutes, srcAddr, "")
	}
	addAddr(&ipContext.DstIpAddrs, v.pool.selfAddress().String())

	conn.GetContext().IpContext = ipContext

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
