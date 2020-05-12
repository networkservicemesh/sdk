package next

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
)

type tailDiscoveryServer struct {
}

func (t tailDiscoveryServer) FindNetworkService(context.Context, *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	return nil, errors.New("NetworkService is not found")
}

var _ registry.NetworkServiceDiscoveryServer = &tailDiscoveryServer{}
