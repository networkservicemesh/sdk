package memory

import (
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/tools/repository"
)

type Memory interface {
	NetworkServiceEndpoints() repository.RegistryNetworkServiceEndpointRepository
	NetworkServiceManagers() repository.RegistryNetworkServiceManagerRepository
	NetworkServices() repository.RegistryNetworkServiceRepository
}

type memory struct {
	networkServiceEndpoints repository.RegistryNetworkServiceEndpointRepository
	networkServiceManagers  repository.RegistryNetworkServiceManagerRepository
	networkServices         repository.RegistryNetworkServiceRepository
}

func (m *memory) NetworkServiceEndpoints() repository.RegistryNetworkServiceEndpointRepository {
	return m.networkServiceEndpoints
}

func (m *memory) NetworkServiceManagers() repository.RegistryNetworkServiceManagerRepository {
	return m.networkServiceManagers
}

func (m *memory) NetworkServices() repository.RegistryNetworkServiceRepository {
	return m.networkServices
}

func NewMemory() Memory {
	return &memory{
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

var _ Memory = &memory{}
