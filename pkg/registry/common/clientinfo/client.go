// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package clientinfo provides a registry client that adds pod, node and cluster names to registration
package clientinfo

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clientinfoutils"
)

type clientInfoRegistry struct{}

// NewRegistryClient - creates a new registry.NetworkServiceRegistryClient that adds pod, node and cluster names
// to registration from corresponding environment variables
func NewRegistryClient() registry.NetworkServiceRegistryClient {
	return &clientInfoRegistry{}
}

func (a *clientInfoRegistry) RegisterNSE(ctx context.Context, in *registry.NSERegistration, opts ...grpc.CallOption) (*registry.NSERegistration, error) {
	nse := in.NetworkServiceEndpoint
	if nse.Labels == nil {
		nse.Labels = make(map[string]string)
	}
	clientinfoutils.AddClientInfo(nse.Labels)
	return next.RegistryClient(ctx).RegisterNSE(ctx, in, opts...)
}

func (a *clientInfoRegistry) BulkRegisterNSE(ctx context.Context, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_BulkRegisterNSEClient, error) {
	return next.RegistryClient(ctx).BulkRegisterNSE(ctx, opts...)
}

func (a *clientInfoRegistry) RemoveNSE(ctx context.Context, in *registry.RemoveNSERequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.RegistryClient(ctx).RemoveNSE(ctx, in, opts...)
}
