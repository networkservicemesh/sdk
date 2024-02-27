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

// Package strictipam provides a networkservice.NetworkService Server chain element for building an IPAM server that prevents IP context configuration out of the settings scope
package strictipam

import (
	"context"
	"net"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/ipam"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
)

type strictIPAMServer struct {
	ipPool *ippool.IPPool
	m      sync.Mutex
}

// NewServer - returns a new ipam networkservice.NetworkServiceServer that validates the incoming IP context parameters and resets them based on the validation result.
func NewServer(ctx context.Context, prefixCh <-chan *ipam.PrefixResponse) networkservice.NetworkServiceServer {
	var ipPool = ippool.New(net.IPv4len)

	s := &strictIPAMServer{ipPool: ipPool}
	go func() {
		for prefix := range prefixCh {
			s.m.Lock()
			s.ipPool = ippool.NewWithNetString(prefix.Prefix)
			s.m.Unlock()
		}
	}()

	return s
}

func (n *strictIPAMServer) areAddressesValid(addresses []string) bool {
	n.m.Lock()
	defer n.m.Unlock()

	for _, addr := range addresses {
		if !n.ipPool.ContainsNetString(addr) {
			return false
		}
	}
	return true
}

func (n *strictIPAMServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	srcAddrs := request.GetConnection().GetContext().GetIpContext().GetSrcIpAddrs()
	dstAddrs := request.GetConnection().GetContext().GetIpContext().GetDstIpAddrs()

	if !n.areAddressesValid(srcAddrs) && !n.areAddressesValid(dstAddrs) {
		request.Connection.Context.IpContext = &networkservice.IPContext{}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (n *strictIPAMServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
