// Copyright (c) 2024 Cisco and its affiliates.
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

// Package strictipam provides a networkservice.NetworkService Server chain element for building an IPAM server that
// filters some invalid addresses and routes in IP context
package strictipam

import (
	"context"
	"net"
	"net/netip"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dualstack"
)

type strictIPAMServer struct {
	ipPool *dualstack.IPPool
}

// NewServer - creates a new strict IPAM server
func NewServer(newIPAMServer func(...*net.IPNet) networkservice.NetworkServiceServer, prefixes ...*net.IPNet) networkservice.NetworkServiceServer {
	if newIPAMServer == nil {
		panic("newIPAMServer should not be nil")
	}
	var ipPool = dualstack.New()
	for _, p := range prefixes {
		ipPool.AddIPNet(p)
	}
	return next.NewNetworkServiceServer(
		&strictIPAMServer{ipPool: ipPool},
		newIPAMServer(prefixes...),
	)
}

func (s *strictIPAMServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	s.validateIPContext(request.Connection.Context.IpContext)
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	s.pullAddrs(conn.Context.IpContext)
	return conn, nil
}

func (s *strictIPAMServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	s.free(conn.Context.IpContext)
	return next.Server(ctx).Close(ctx, conn)
}

func (s *strictIPAMServer) getInvalidAddrs(addrs []string) []string {
	invalidAddrs := make([]string, 0)
	for _, prefixString := range addrs {
		prefix, parseErr := netip.ParsePrefix(prefixString)
		if parseErr != nil {
			invalidAddrs = append(invalidAddrs, prefixString)
			continue
		}

		if !s.ipPool.ContainsIPString(prefix.Addr().String()) {
			invalidAddrs = append(invalidAddrs, prefixString)
		}
	}

	return invalidAddrs
}

func (s *strictIPAMServer) validateIPContext(ipContext *networkservice.IPContext) {
	for _, addr := range s.getInvalidAddrs(ipContext.SrcIpAddrs) {
		deleteAddr(&ipContext.SrcIpAddrs, addr)
		deleteRoute(&ipContext.DstRoutes, addr)
	}

	for _, addr := range s.getInvalidAddrs(ipContext.DstIpAddrs) {
		deleteAddr(&ipContext.DstIpAddrs, addr)
		deleteRoute(&ipContext.SrcRoutes, addr)
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

func deleteAddr(addrs *[]string, addr string) {
	for i, a := range *addrs {
		if a == addr {
			*addrs = append((*addrs)[:i], (*addrs)[i+1:]...)
			return
		}
	}
}

func (s *strictIPAMServer) pullAddrs(ipContext *networkservice.IPContext) {
	for _, addr := range ipContext.SrcIpAddrs {
		_, _ = s.ipPool.PullIPString(addr)
	}

	for _, addr := range ipContext.DstIpAddrs {
		_, _ = s.ipPool.PullIPString(addr)
	}
}

func (s *strictIPAMServer) free(ipContext *networkservice.IPContext) {
	for _, addr := range ipContext.SrcIpAddrs {
		_, ipNet, err := net.ParseCIDR(addr)
		if err != nil {
			return
		}
		s.ipPool.AddIPNet(ipNet)
	}

	for _, addr := range ipContext.DstIpAddrs {
		_, ipNet, err := net.ParseCIDR(addr)
		if err != nil {
			return
		}
		s.ipPool.AddIPNet(ipNet)
	}
}
