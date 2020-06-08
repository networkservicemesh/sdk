package next

import (
	"context"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type tailDiscoveryClient struct {
}

func (t *tailDiscoveryClient) FindNetworkService(_ context.Context, request *registry.FindNetworkServiceRequest, _ ...grpc.CallOption) (*registry.FindNetworkServiceResponse, error) {
	// Return empty
	return &registry.FindNetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name: request.NetworkServiceName,
		},
	}, nil
}

var _ registry.NetworkServiceDiscoveryClient = &tailDiscoveryClient{}
