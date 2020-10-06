// Copyright (c) 2020 Cisco and/or its affiliates.
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
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type kernelMechanismClient struct {
	interfaceName string
}

// NewClient - returns client that sets kernel preferred mechanism
func NewClient(options ...Option) networkservice.NetworkServiceClient {
	k := &kernelMechanismClient{}
	for _, opt := range options {
		opt(k)
	}
	return k
}

func (k *kernelMechanismClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// Append the mechanism preference to the request
	preferredMechanism := &networkservice.Mechanism{
		Cls:  cls.LOCAL,
		Type: kernel.MECHANISM,
		Parameters: map[string]string{
			kernel.NetNSURL:         (&url.URL{Scheme: "file", Path: netNSFilename}).String(),
			kernel.InterfaceNameKey: k.interfaceName,
		},
	}
	request.MechanismPreferences = append(request.GetMechanismPreferences(), preferredMechanism)
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (k *kernelMechanismClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

// Option for kernel mechanism client
type Option func(k *kernelMechanismClient)

// WithInterfaceName sets interface name
func WithInterfaceName(interfaceName string) Option {
	return func(k *kernelMechanismClient) {
		k.interfaceName = interfaceName
	}
}
