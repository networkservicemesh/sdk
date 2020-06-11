package adapters

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"google.golang.org/grpc"
)

type networkServiceRegistryClient struct {
	server registry.NetworkServiceRegistryServer
}

func (n *networkServiceRegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEntry, _ ...grpc.CallOption) (*registry.NetworkServiceEntry, error) {
	return n.server.Register(ctx, in)
}

func (n *networkServiceRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, _ ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	ch := make(chan *registry.NetworkServiceEntry)
	s := streamchannel.NewNetworkServiceFindServer(ctx, ch)
	if err := n.server.Find(in, s); err != nil {
		return nil, err
	}
	return streamchannel.NewNetworkServiceFindClient(s.Context(), ch), nil
}

func (n *networkServiceRegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEntry, opts ...grpc.CallOption) (*empty.Empty, error) {
	return n.server.Unregister(ctx, in)
}

var _ registry.NetworkServiceRegistryClient = &networkServiceRegistryClient{}

// RegistryServerToClient - returns a registry.NetworkServiceRegistryServer wrapped around the supplied server
func RegistryServerToClient(server registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryClient {
	return &networkServiceRegistryClient{server: server}
}
