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

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
)

type vniClient struct {
	tunnelIP net.IP
}

// NewClient - set the SrcIP for the vxlan mechanism
func NewClient(tunnelIP net.IP) networkservice.NetworkServiceClient {
	return &vniClient{
		tunnelIP: tunnelIP,
	}
}

func (v *vniClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	for _, m := range request.GetMechanismPreferences() {
		// Note: This only has effect if this is a vxlan mechanism
		if mech := vxlan.ToMechanism(m); mech != nil {
			mech.SetSrcIP(v.tunnelIP)
		}
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (v *vniClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
