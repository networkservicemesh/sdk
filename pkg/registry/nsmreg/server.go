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

// Package nsmreg - provide Network Service Mesh manager registry tracker, to update endpoint registrations.
package nsmreg

import (
	"github.com/networkservicemesh/sdk/pkg/registry/common/setmgr"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

// NewNSMRegServer - construct a new NSM aware NSE registry server
func NewNSMRegServer(manager *registry.NetworkServiceManager, storage *memory.Storage, nsmRegistryClient registry.NsmRegistryClient) registry.NsmRegistryServer {
	return newNSMregistryServer(
		newNSMRegistryClientToServer(nsmRegistryClient),
		memory.NewNSMRegistryServer(manager.Name, storage),
	)
}

// NewNetworkServiceRegistryServer - construct a new NSM aware NSE registry server
func NewNetworkServiceRegistryServer(manager *registry.NetworkServiceManager, storage *memory.Storage, localbypassRegistry registry.NetworkServiceRegistryServer, registryClient registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryServer {
	return newRegistryServer(
		setmgr.NewServer(manager), // Manager will be updated here
		newRegistryClientToServer(registryClient),
		memory.NewNetworkServiceRegistryServer(storage),
		localbypassRegistry,
	)
}

// NewNetworkServiceDiscoveryServer - construct a new NSM aware Discovery server
func NewNetworkServiceDiscoveryServer(manager *registry.NetworkServiceManager, storage *memory.Storage, discoveryClient registry.NetworkServiceDiscoveryClient) registry.NetworkServiceDiscoveryServer {
	return newDiscoveryServer(
		newDiscoveryClientToServer(discoveryClient),
		memory.NewNetworkServiceDiscoveryServer(storage),
	)
}
