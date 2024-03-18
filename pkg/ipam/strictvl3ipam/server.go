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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/ipcontext/vl3"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type strictVl3IPAMServer struct {
	vl3IPAMs []*vl3.IPAM
}

// NewServer - returns a new ipam networkservice.NetworkServiceServer that validates the incoming IP context parameters and resets them based on the validation result.
func NewServer(ctx context.Context, newVl3IPAMServer func(context.Context, *vl3.IPAM) networkservice.NetworkServiceServer, vl3IPAMs ...*vl3.IPAM) networkservice.NetworkServiceServer {
	elements := []networkservice.NetworkServiceServer{&strictVl3IPAMServer{vl3IPAMs: vl3IPAMs}}
	for _, ipam := range vl3IPAMs {
		elements = append(elements, newVl3IPAMServer(ctx, ipam))
	}
	return next.NewNetworkServiceServer(elements...)
}

func (s *strictVl3IPAMServer) areAddressesValid(addresses []string) bool {
	if len(addresses) == 0 {
		return true
	}

	for _, addr := range addresses {
		for _, ipam := range s.vl3IPAMs {
			if ipam.ContainsNetString(addr) {
				return true
			}
		}
	}
	return false
}

func (s *strictVl3IPAMServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if !s.areAddressesValid(request.GetConnection().GetContext().GetIpContext().GetDstIpAddrs()) {
		request.Connection.Context.IpContext = &networkservice.IPContext{}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (s *strictVl3IPAMServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
