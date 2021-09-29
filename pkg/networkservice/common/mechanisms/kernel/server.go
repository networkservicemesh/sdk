// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

// Package kernel provides the necessary mechanisms to request and inject a kernel interface.
package kernel

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

type kernelMechanismServer struct {
	interfaceName string
	freeVLANs     *roaring.Bitmap
	lock          sync.Mutex
}

// NewServer - creates a NetworkServiceServer that requests a kernel interface and populates the netns inode
func NewServer(opts ...Option) networkservice.NetworkServiceServer {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	vlans := roaring.New()
	vlans.AddRange(1, 4095)
	return &kernelMechanismServer{
		interfaceName: o.interfaceName,
		freeVLANs:     vlans,
	}
}

func (m *kernelMechanismServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	mechanism := kernelmech.ToMechanism(request.GetConnection().GetMechanism())
	if mechanism != nil {
		mechanism.SetNetNSURL(netNSURL)
		if m.interfaceName != "" {
			mechanism.SetInterfaceName(m.interfaceName)
		} else {
			mechanism.SetInterfaceName(getNameFromConnection(request.GetConnection()))
		}
		if mechanism.SupportsVLAN() {
			m.lock.Lock()
			if m.freeVLANs.IsEmpty() {
				m.lock.Unlock()
				return nil, errors.New("vlan id pool is empty")
			}
			vlanID := m.freeVLANs.Minimum()
			m.freeVLANs.Remove(vlanID)
			mechanism.SetVLAN(vlanID)
			m.lock.Unlock()
		}
	}
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil && mechanism != nil && mechanism.SupportsVLAN() {
		m.releaseVLANID(mechanism)
	}
	return conn, err
}

func (m *kernelMechanismServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	mechanism := kernelmech.ToMechanism(conn.GetMechanism())
	if mechanism != nil && mechanism.SupportsVLAN() {
		m.releaseVLANID(mechanism)
	}
	return next.Server(ctx).Close(ctx, conn)
}

func (m *kernelMechanismServer) releaseVLANID(mechanism *kernelmech.Mechanism) {
	m.lock.Lock()
	defer m.lock.Unlock()
	vlanID := mechanism.GetVLAN()
	if vlanID > 0 {
		m.freeVLANs.Add(vlanID)
	}
}
