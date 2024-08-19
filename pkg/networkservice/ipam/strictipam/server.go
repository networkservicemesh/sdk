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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dualstackippool"
)

type strictIPAMServer struct {
	ipPool *dualstackippool.DualStackIPPool
}

// NewServer - returns a new ipam networkservice.NetworkServiceServer that validates the incoming IP context parameters and resets them based on the validation result.
func NewServer(newIPAMServer func(...*net.IPNet) networkservice.NetworkServiceServer, prefixes ...*net.IPNet) networkservice.NetworkServiceServer {
	if newIPAMServer == nil {
		panic("newIPAMServer should not be nil")
	}
	var ipPool = dualstackippool.New()
	for _, p := range prefixes {
		ipPool.AddNet(p)
	}
	return next.NewNetworkServiceServer(
		&strictIPAMServer{ipPool: ipPool},
		newIPAMServer(prefixes...),
	)
}

func (n *strictIPAMServer) areAddressesValid(addresses []string) bool {
	for _, srcIP := range addresses {
		if !n.ipPool.ContainsString(srcIP) {
			return false
		}
	}
	return true
}

func (n *strictIPAMServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if !n.areAddressesValid(request.GetConnection().GetContext().GetIpContext().GetSrcIpAddrs()) ||
		!n.areAddressesValid(request.GetConnection().GetContext().GetIpContext().GetDstIpAddrs()) {
		request.GetConnection().GetContext().IpContext = &networkservice.IPContext{}
	}

	return next.Server(ctx).Request(ctx, request)
}

func (n *strictIPAMServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
