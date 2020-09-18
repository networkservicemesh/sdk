package memory_floating

import (
	"github.com/networkservicemesh/sdk/pkg/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
)

func NewServer() registry.Registry {
	return registry.NewServer(
		chain.NewNetworkServiceRegistryServer(memory.NewNetworkServiceRegistryServer()),
		chain.NewNetworkServiceEndpointRegistryServer(memory.NewNetworkServiceEndpointRegistryServer()),
	)
}
