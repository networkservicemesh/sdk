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

//go:build linux
// +build linux

// Package vlan provides server chain element setting vlan id on the kernel mechanism
package vlan

import (
	"context"
	"errors"
	"sync"

	"github.com/RoaringBitmap/roaring"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type vlanServer struct {
	freeVLANs *roaring.Bitmap
	lock      sync.Mutex
}

// NewServer - creates a NetworkServiceServer that requests a kernel interface with vlan parameter
func NewServer() networkservice.NetworkServiceServer {
	vlans := roaring.New()
	vlans.AddRange(1, 4095)
	return &vlanServer{
		freeVLANs: vlans,
	}
}

func (m *vlanServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	mechanism := kernelmech.ToMechanism(request.GetConnection().GetMechanism())
	isEstablished := true
	if mechanism != nil && mechanism.SupportsVLAN() && mechanism.GetVLAN() == 0 {
		m.lock.Lock()
		if m.freeVLANs.IsEmpty() {
			m.lock.Unlock()
			return nil, errors.New("vlan id not available for allocation")
		}
		vlanID := m.freeVLANs.Minimum()
		m.freeVLANs.Remove(vlanID)
		m.lock.Unlock()

		mechanism.SetVLAN(vlanID)
		isEstablished = false
	}
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil && mechanism != nil && !isEstablished {
		m.releaseVLANID(mechanism)
	}
	return conn, err
}

func (m *vlanServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	mechanism := kernelmech.ToMechanism(conn.GetMechanism())
	if mechanism != nil && mechanism.SupportsVLAN() && mechanism.GetVLAN() > 0 {
		m.releaseVLANID(mechanism)
	}
	return next.Server(ctx).Close(ctx, conn)
}

func (m *vlanServer) releaseVLANID(mechanism *kernelmech.Mechanism) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.freeVLANs.Add(mechanism.GetVLAN())
}
