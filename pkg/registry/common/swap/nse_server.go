package swap

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type swapNSEServer struct {
	currentDomain string
}

func (s *swapNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	nse.Name = fmt.Sprintf("%v@%v", nse.Name, s.currentDomain)
	addr := connect.Addr(ctx)
	if addr == nil {
		return nil, errors.New("address to swap has not passed")
	}
	nse.Url = fmt.Sprintf("tcp://%v", connect.Addr(ctx))
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *swapNSEServer) Find(q *registry.NetworkServiceEndpointQuery, srv registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(srv.Context()).Find(q, srv)
}

func (s *swapNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	nse.Name = fmt.Sprintf("%v@%v", nse.Name, s.currentDomain)
	addr := connect.Addr(ctx)
	if addr == nil {
		return nil, errors.New("address to swap has not passed")
	}
	nse.Url = fmt.Sprintf("tcp://%v", connect.Addr(ctx))

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

func NewNetworkServiceEndpointServer() registry.NetworkServiceEndpointRegistryServer {
	return &swapNSEServer{}
}
