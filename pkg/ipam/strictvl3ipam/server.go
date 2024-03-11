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

// Package strictvl3ipam provides a networkservice.NetworkService Server chain element that resets IP context configuration out of the settings scope
package strictvl3ipam

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

type strictVl3IPAMServer struct {
	ipPool *ippool.IPPool
	m      sync.Mutex
}

// NewServer - returns a new ipam networkservice.NetworkServiceServer that validates the incoming IP context parameters and resets them based on the validation result.
func NewServer(ctx context.Context, newVl3IPAMServer func(ctx context.Context, prefixCh <-chan *ipam.PrefixResponse) networkservice.NetworkServiceServer, prefixCh <-chan *ipam.PrefixResponse) networkservice.NetworkServiceServer {
	var ipPool = ippool.New(net.IPv6len)

	vl3IPAMPrefixCh := make(chan *ipam.PrefixResponse, 1)
	server := &strictVl3IPAMServer{ipPool: ipPool}
	go func() {
		defer close(vl3IPAMPrefixCh)
		for prefix := range prefixCh {
			vl3IPAMPrefixCh <- prefix
			server.m.Lock()
			server.ipPool.Clear()
			server.ipPool.AddNetString(prefix.Prefix)
			server.m.Unlock()
		}
	}()

	return next.NewNetworkServiceServer(
		server,
		newVl3IPAMServer(ctx, vl3IPAMPrefixCh))
}

func (s *strictVl3IPAMServer) areAddressesValid(addresses []string) bool {
	s.m.Lock()
	defer s.m.Unlock()

	if len(addresses) == 0 {
		return true
	}

	for _, addr := range addresses {
		if s.ipPool.ContainsNetString(addr) {
			return true
		}
	}
	return false
}

func (s *strictVl3IPAMServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	srcAddrs := request.GetConnection().GetContext().GetIpContext().GetSrcIpAddrs()
	dstAddrs := request.GetConnection().GetContext().GetIpContext().GetDstIpAddrs()

	if !s.areAddressesValid(srcAddrs) || !s.areAddressesValid(dstAddrs) {
		request.Connection.Context.IpContext = &networkservice.IPContext{}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (s *strictVl3IPAMServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
