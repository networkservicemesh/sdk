// Copyright (c) 2020-2021 Cisco and/or its affiliates.
//
// Copyright (c) 2021 Nordix Foundation.
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

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
)

type vniClient struct {
	tunnelIP   net.IP
	tunnelPort uint16
}

// NewClient - set the SrcIP for the vxlan mechanism.
func NewClient(tunnelIP net.IP, options ...Option) networkservice.NetworkServiceClient {
	opts := &vniOpions{
		tunnelPort: vxlanPort,
	}
	for _, opt := range options {
		opt(opts)
	}

	return &vniClient{
		tunnelIP:   tunnelIP,
		tunnelPort: opts.tunnelPort,
	}
}

func (v *vniClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	for _, m := range request.GetMechanismPreferences() {
		// Note: This only has effect if this is a vxlan mechanism
		if mech := vxlan.ToMechanism(m); mech != nil {
			mech.SetSrcIP(v.tunnelIP)
			mech.SetSrcPort(v.tunnelPort)

			log.FromContext(ctx).
				WithField("VNIclient", "request").
				WithField("mechSrcIp", mech.SrcIP()).
				WithField("mechSrcPort", mech.SrcPort()).
				Debugf("set mechanism src")
		}
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (v *vniClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
