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

// Package nsmgr - provide Network Service Mesh manager registry tracker, to update endpoint registrations.
package nsmgr

import (
	"context"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/custom"
	"github.com/networkservicemesh/sdk/pkg/registry/common/localbypass"
	adapter_registry "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	chain_registry "github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

// Registry - a registry interface combining all 3 registry interfaces
type Registry interface {
	registry.NetworkServiceRegistryServer
	registry.NetworkServiceDiscoveryServer
	registry.NsmRegistryServer

	localbypass.LocalBypass
}

type nsmgrRegistrySever struct {
	registry.NsmRegistryServer
	registry.NetworkServiceRegistryServer
	registry.NetworkServiceDiscoveryServer
	localbypass.LocalBypass
	storage memory.Storage
}

// NewServer - construct a new NSM aware NSE registry server
// Server is using latest NSM registration and update endpoints accordingly.
// if registryClient, nsmRegistryClient, discoveryClient are nil, they will be skipped.
func NewServer(manager *registry.NetworkServiceManager,
	registryClient registry.NetworkServiceRegistryClient,
	nsmRegistryClient registry.NsmRegistryClient,
	discoveryClient registry.NetworkServiceDiscoveryClient) Registry {
	rv := &nsmgrRegistrySever{}

	// ----------------------------------------------------------------
	// -----D-E-F-I-N-E----- NsmRegistryServer
	// ----------------------------------------------------------------
	rv.NsmRegistryServer = chain_registry.NewNSMRegistryServer(
		adapter_registry.NewNSMClientToServer(nsmRegistryClient),
		memory.NewNSMRegistryServer(manager.Name, &rv.storage),
	)

	// ----------------------------------------------------------------
	// -----D-E-F-I-N-E----- NetworkServiceRegistryServer
	// ----------------------------------------------------------------
	localBypassRegistry := localbypass.NewServer()
	rv.LocalBypass = localBypassRegistry

	// define NSE registry server
	updateNSM := func(ctx context.Context, registration *registry.NSERegistration) (*registry.NSERegistration, error) {
		registration.NetworkServiceManager = manager
		return registration, nil
	}

	rv.NetworkServiceRegistryServer = chain_registry.NewNetworkServiceRegistryServer(
		custom.NewServer(updateNSM, updateNSM, nil), // Manager will be updated after return
		localBypassRegistry,
		adapter_registry.NewRegistryClientToServer(registryClient),
		memory.NewNetworkServiceRegistryServer(&rv.storage),
	)

	// ----------------------------------------------------------------
	// -----D-E-F-I-N-E----- NetworkServiceDiscoveryServer
	// ----------------------------------------------------------------
	rv.NetworkServiceDiscoveryServer = chain_registry.NewDiscoveryServer(
		adapter_registry.NewDiscoveryClientToServer(discoveryClient),
		memory.NewNetworkServiceDiscoveryServer(&rv.storage),
	)

	return rv
}

// Register - register registry to GRPC server
func (n *nsmgrRegistrySever) Register(s *grpc.Server) {
	registry.RegisterNsmRegistryServer(s, n)
	registry.RegisterNetworkServiceRegistryServer(s, n)
	registry.RegisterNetworkServiceDiscoveryServer(s, n)
}
