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

package memory

import (
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/repository"
)

// ResourcesClient provides API for access to NSM resources
type ResourcesClient interface {
	// NetworkServiceEndpoints returns interface for access to Network Service Endpoints
	NetworkServiceEndpoints() repository.RegistryNetworkServiceEndpointRepository
	// NetworkServiceManagers returns interface for access to Network Service Managers
	NetworkServiceManagers() repository.RegistryNetworkServiceManagerRepository
	// NetworkServiceManagers returns interface for access to Network Services
	NetworkServices() repository.RegistryNetworkServiceRepository
}

type resourcesClient struct {
	networkServiceEndpoints repository.RegistryNetworkServiceEndpointRepository
	networkServiceManagers  repository.RegistryNetworkServiceManagerRepository
	networkServices         repository.RegistryNetworkServiceRepository
}

func (m *resourcesClient) NetworkServiceEndpoints() repository.RegistryNetworkServiceEndpointRepository {
	return m.networkServiceEndpoints
}

func (m *resourcesClient) NetworkServiceManagers() repository.RegistryNetworkServiceManagerRepository {
	return m.networkServiceManagers
}

func (m *resourcesClient) NetworkServices() repository.RegistryNetworkServiceRepository {
	return m.networkServices
}

// NewMemoryResourceClient creates new ResourcesClient instance based on resourceClient
func NewMemoryResourceClient() ResourcesClient {
	return &resourcesClient{
		networkServices: repository.NewRegistryNetworkServiceMemoryRepository(func(service *registry.NetworkService) string {
			return service.Name
		}),
		networkServiceManagers: repository.NewRegistryNetworkServiceManagerMemoryRepository(func(manager *registry.NetworkServiceManager) string {
			return manager.Name
		}),
		networkServiceEndpoints: repository.NewRegistryNetworkServiceEndpointMemoryRepository(func(endpoint *registry.NetworkServiceEndpoint) string {
			return endpoint.Name
		}),
	}
}

var _ ResourcesClient = &resourcesClient{}
