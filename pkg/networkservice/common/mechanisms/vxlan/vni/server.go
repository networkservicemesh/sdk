// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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

package vni

import (
	"context"
	"net"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type vniServer struct {
	tunnelIP net.IP
	sync.Map
}

// NewServer - set the DstIP *and* VNI for the vxlan mechanism
func NewServer(tunnelIP net.IP) networkservice.NetworkServiceServer {
	return &vniServer{
		tunnelIP: tunnelIP,
	}
}

func (v *vniServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if mechanism := vxlan.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil {
		mechanism.SetDstIP(v.tunnelIP)
		k := vniKey{
			srcIPString: mechanism.SrcIP().String(),
			vni:         mechanism.VNI(),
		}
		// If we already have a VNI, make sure we remember it, and go on
		if k.vni != 0 && mechanism.SrcIP() != nil {
			v.Map.LoadOrStore(k, &k)
			return next.Server(ctx).Request(ctx, request)
		}

		for {
			// Generate a random VNI (appropriately odd or even)
			var err error
			k.vni, err = mechanism.GenerateRandomVNI()
			if err != nil {
				return nil, err
			}
			// If its not one already in use, set it and we are good to go
			if _, ok := v.Map.LoadOrStore(k, &k); !ok {
				mechanism.SetVNI(k.vni)
				break
			}
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (v *vniServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if mechanism := vxlan.ToMechanism(conn.GetMechanism()); mechanism != nil {
		k := vniKey{
			srcIPString: mechanism.SrcIP().String(),
			vni:         mechanism.VNI(),
		}
		if k.vni != 0 && mechanism.SrcIP() != nil {
			v.Map.LoadAndDelete(k)
		}
	}
	return next.Server(ctx).Close(ctx, conn)
}
