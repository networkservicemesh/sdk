// Copyright (c) 2020-2021 Cisco and/or its affiliates.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package kernel

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/nanoid"
)

type kernelMechanismClient struct {
	interfaceName          string
	interfaceNameGenerator func(ns string) (string, error)
}

// NewClient - returns client that sets kernel preferred mechanism.
func NewClient(opts ...Option) networkservice.NetworkServiceClient {
	o := &options{
		interfaceNameGenerator: nanoid.GenerateLinuxInterfaceName,
	}
	for _, opt := range opts {
		opt(o)
	}
	return &kernelMechanismClient{
		interfaceName:          o.interfaceName,
		interfaceNameGenerator: o.interfaceNameGenerator,
	}
}

func (k *kernelMechanismClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if err := k.manageMechanismPreferences(request); err != nil {
		return nil, errors.Wrap(err, "Failed to update mechanism preferences")
	}

	return next.Client(ctx).Request(ctx, request, opts...)
}

func (k *kernelMechanismClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

// manageMechanismPreferences updates mechanism preferences. Adds kernel mechanism if they doesn't have one.
func (k *kernelMechanismClient) manageMechanismPreferences(request *networkservice.NetworkServiceRequest) error {
	updated := false
	for _, m := range request.GetRequestMechanismPreferences() {
		if mechanism := kernelmech.ToMechanism(m); mechanism != nil {
			if err := k.updateMechanism(request.GetConnection().GetNetworkService(), mechanism); err != nil {
				return err
			}
			mechanism.SetNetNSURL(netNSURL)
			updated = true
		}
	}

	if updated {
		return nil
	}

	mechanism := kernelmech.ToMechanism(kernelmech.New(netNSURL))
	if err := k.updateMechanism(request.GetConnection().GetNetworkService(), mechanism); err != nil {
		return err
	}

	request.MechanismPreferences = append(request.GetMechanismPreferences(), mechanism.Mechanism)
	return nil
}

func (k *kernelMechanismClient) updateMechanism(ns string, mechanism *kernelmech.Mechanism) error {
	if mechanism.GetInterfaceName() == "" {
		if k.interfaceName != "" {
			mechanism.SetInterfaceName(k.interfaceName)
		} else {
			ifname, err := k.interfaceNameGenerator(ns)
			if err != nil {
				return errors.Wrap(err, "Failed to generate kernel interface name")
			}
			mechanism.SetInterfaceName(ifname)
		}
	}

	return nil
}
