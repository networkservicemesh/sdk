package streamchannel

import (
	"context"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
)

func NewNetworkServiceFindServer(ctx context.Context, sendCh chan<- *registry.NetworkServiceEntry) registry.NetworkServiceRegistry_FindServer {
	return &networkServiceEndpointRegistryFindServer{
		ctx:    ctx,
		sendCh: sendCh,
	}
}

type networkServiceEndpointRegistryFindServer struct {
	grpc.ServerStream
	ctx    context.Context
	sendCh chan<- *registry.NetworkServiceEntry
}

func (s *networkServiceEndpointRegistryFindServer) Send(entry *registry.NetworkServiceEntry) error {
	s.sendCh <- entry
	return nil
}

func (s *networkServiceEndpointRegistryFindServer) Context() context.Context {
	return s.ctx
}

var _ registry.NetworkServiceRegistry_FindServer = &networkServiceEndpointRegistryFindServer{}
