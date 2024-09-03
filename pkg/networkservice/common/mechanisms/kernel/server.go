// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2024 Cisco and/or its affiliates.
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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/nanoid"
)

type kernelMechanismServer struct {
	interfaceName          string
	interfaceNameGenerator func(string) (string, error)
}

// NewServer - creates a NetworkServiceServer that requests a kernel interface and populates the netns inode.
func NewServer(opts ...Option) networkservice.NetworkServiceServer {
	o := &options{
		interfaceNameGenerator: nanoid.GenerateLinuxInterfaceName,
	}
	for _, opt := range opts {
		opt(o)
	}
	return &kernelMechanismServer{
		interfaceName:          o.interfaceName,
		interfaceNameGenerator: o.interfaceNameGenerator,
	}
}

func (m *kernelMechanismServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if mechanism := kernelmech.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil {
		mechanism.SetNetNSURL(netNSURL)
		if mechanism.GetInterfaceName() == "" {
			if m.interfaceName != "" {
				mechanism.SetInterfaceName(m.interfaceName)
			} else {
				ifname, err := m.interfaceNameGenerator(request.GetConnection().GetNetworkService())
				if err != nil {
					return nil, errors.Wrap(err, "Failed to generate kernel interface name")
				}
				mechanism.SetInterfaceName(ifname)
			}
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (m *kernelMechanismServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
