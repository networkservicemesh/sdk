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

// Package checkregistration - provides registry chain elements to check the registration received from the
// previous element in the chain
package checkregistration

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type checkRegistrationRegistryClient struct {
	*testing.T
	check func(*testing.T, *registry.NSERegistration)
}

// NewRegistryClient - returns registry.NetworkServiceRegistryClient chain elements to check the registration
// received from the previous element in the chain
//             t - *testing.T for checks
//             check - function to check the registry.NSERegistration
func NewRegistryClient(t *testing.T, check func(*testing.T, *registry.NSERegistration)) registry.NetworkServiceRegistryClient {
	return &checkRegistrationRegistryClient{t, check}
}

func (c *checkRegistrationRegistryClient) RegisterNSE(ctx context.Context, registration *registry.NSERegistration, opts ...grpc.CallOption) (*registry.NSERegistration, error) {
	c.check(c.T, registration)
	return next.RegistryClient(ctx).RegisterNSE(ctx, registration, opts...)
}

func (c *checkRegistrationRegistryClient) BulkRegisterNSE(ctx context.Context, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_BulkRegisterNSEClient, error) {
	return next.RegistryClient(ctx).BulkRegisterNSE(ctx, opts...)
}

func (c *checkRegistrationRegistryClient) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.RegistryClient(ctx).RemoveNSE(ctx, request, opts...)
}
