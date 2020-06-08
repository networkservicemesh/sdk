package next

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type tailDiscoveryServer struct {
}

func (t tailDiscoveryServer) FindNetworkService(_ context.Context, request *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	// Return empty
	return &registry.FindNetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name: request.NetworkServiceName,
		},
	}, nil
}

var _ registry.NetworkServiceDiscoveryServer = &tailDiscoveryServer{}
