package chain

import (
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

// NewNetworkServiceRegistryServer - creates a chain of servers
func NewNetworkServiceRegistryServer(servers ...registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	return next.NewWrappedNetworkServiceRegistryServer(func(server registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
		return server
	}, servers...)
}

// NewNetworkServiceRegistryClient - creates a chain of clients
func NewNetworkServiceRegistryClient(clients ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return next.NewWrappedNetworkServiceRegistryClient(func(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
		return client
	}, clients...)
}
