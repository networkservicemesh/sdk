package next

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
)

type tailDiscoveryClient struct {
}

func (t tailDiscoveryClient) FindNetworkService(context.Context, *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	return nil, errors.New("NetworkService is not found")
}

var _ registry.NetworkServiceDiscoveryServer = &tailDiscoveryClient{}
