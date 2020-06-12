package chain

import (
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

// NewNetworkServiceEndpointRegistryServer - creates a chain of servers
func NewNetworkServiceEndpointRegistryServer(servers ...registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
	return next.NewWrappedNetworkServiceEndpointRegistryServer(func(server registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
		return server
	}, servers...)
}

// NewNetworkServiceEndpointRegistryClient - creates a chain of clients
func NewNetworkServiceEndpointRegistryClient(clients ...registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
	return next.NewWrappedNetworkServiceEndpointRegistryClient(func(client registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
		return client
	}, clients...)
}
