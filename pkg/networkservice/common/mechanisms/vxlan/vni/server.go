// Copyright (c) 2020-2021 Cisco and/or its affiliates.
//
// Copyright (c) 2021 Nordix Foundation.
//
// Copyright (c) 2023 Cisco Systems, Inc.
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

	"github.com/edwarnicke/genericsync"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type vniKey struct {
	srcIPString string
	vni         uint32
}

type vniServer struct {
	tunnelIP   net.IP
	tunnelPort uint16

	// This map stores all generated VNIs
	genericsync.Map[vniKey, *vniKey]
}

// NewServer - set the DstIP *and* VNI for the vxlan mechanism.
func NewServer(tunnelIP net.IP, options ...Option) networkservice.NetworkServiceServer {
	opts := &vniOpions{
		tunnelPort: vxlanPort,
	}
	for _, opt := range options {
		opt(opts)
	}

	return &vniServer{
		tunnelIP:   tunnelIP,
		tunnelPort: opts.tunnelPort,
	}
}

func (v *vniServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("VNIserver", "request")

	mechanism := vxlan.ToMechanism(request.GetConnection().GetMechanism())
	if mechanism == nil {
		logger.Debugf("mechanism is not vxlan")
		return next.Server(ctx).Request(ctx, request)
	}
	mechanism.SetDstIP(v.tunnelIP)
	mechanism.SetDstPort(v.tunnelPort)

	logger.WithField("mechanism.DstIP", mechanism.DstIP()).WithField("mechanism.DstPort", mechanism.DstPort()).Debugf("set mechanism dst")

	k := vniKey{
		srcIPString: mechanism.SrcIP().String(),
		vni:         mechanism.VNI(),
	}

	// If we already have a VNI, make sure we remember it, and go on
	if k.vni != 0 && mechanism.SrcIP() != nil {
		_, _ = v.Map.LoadOrStore(k, &k)
		_, loaded := loadOrStore(ctx, metadata.IsClient(v), k.vni)

		logger.WithField("vni", k.vni).Debugf("loadOrStore vni in metadata")

		conn, err := next.Server(ctx).Request(ctx, request)
		if err != nil && !loaded {
			delete(ctx, metadata.IsClient(v))
			v.Map.Delete(k)

			logger.WithField("vni", k.vni).Errorf("error returned from request, deleting vni. err=%v", err.Error())
		}
		return conn, err
	}

	vni, loaded := load(ctx, metadata.IsClient(v))
	if loaded {
		mechanism.SetVNI(vni)
		logger.WithField("vni", vni).Debugf("vni loaded from metadata")
	} else {
		for {
			// Generate a random VNI (appropriately odd or even)
			var err error
			k.vni, err = mechanism.GenerateRandomVNI()
			if err != nil {
				return nil, errors.Wrap(err, "failed to generate a random VNI")
			}
			// If its not one already in use, set it and we are good to go
			if _, ok := v.Map.LoadOrStore(k, &k); !ok {
				mechanism.SetVNI(k.vni)
				store(ctx, metadata.IsClient(v), k.vni)
				logger.WithField("vni", k.vni).Debugf("vni generated and stored in metadata")
				break
			}
		}
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil && !loaded {
		delete(ctx, metadata.IsClient(v))
		v.Map.Delete(k)

		logger.WithField("vniKey.vni", k.vni).Errorf("error returned from request, deleting vni. err=%v", err.Error())
	}
	return conn, err
}

func (v *vniServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if mechanism := vxlan.ToMechanism(conn.GetMechanism()); mechanism != nil {
		k := vniKey{
			srcIPString: mechanism.SrcIP().String(),
			vni:         mechanism.VNI(),
		}

		if k.vni != 0 && mechanism.SrcIP() != nil {
			delete(ctx, metadata.IsClient(v))
			v.Map.LoadAndDelete(k)

			log.FromContext(ctx).
				WithField("VNIserver", "close").
				WithField("vniKey.srcIPString", k.srcIPString).
				WithField("vniKey.vni", k.vni).
				Debugf("vniKey deleted")
		}
	}
	return next.Server(ctx).Close(ctx, conn)
}
