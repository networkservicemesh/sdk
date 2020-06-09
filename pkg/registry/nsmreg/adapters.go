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

package nsmreg

import (
	"github.com/networkservicemesh/api/pkg/api/registry"

	adapter_registry "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next/clients"
)

func newNSMRegistryClientToServer(nsmRegistryClient registry.NsmRegistryClient) registry.NsmRegistryServer {
	var adaptedNsmgrServer registry.NsmRegistryServer
	if nsmRegistryClient != nil {
		adaptedNsmgrServer = adapter_registry.NewNSMClientToServer(clients.NewNextNSMRegistryClient(nsmRegistryClient))
	}
	return adaptedNsmgrServer
}
func newRegistryClientToServer(registryClient registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryServer {
	var adaptedRegistryServer registry.NetworkServiceRegistryServer
	if registryClient != nil {
		adaptedRegistryServer = adapter_registry.NewRegistryClientToServer(clients.NewNextRegistryClient(registryClient))
	}
	return adaptedRegistryServer
}

func newDiscoveryClientToServer(discoveryClient registry.NetworkServiceDiscoveryClient) registry.NetworkServiceDiscoveryServer {
	var adaptedDiscoveryServer registry.NetworkServiceDiscoveryServer
	if discoveryClient != nil {
		adaptedDiscoveryServer = adapter_registry.NewDiscoveryClientToServer(clients.NewNextDiscoveryClient(discoveryClient))
	}
	return adaptedDiscoveryServer
}
