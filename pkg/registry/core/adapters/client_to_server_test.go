package adapters_test

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"testing"
)

type echoClient struct{}

func (t *echoClient) Register(ctx context.Context, in *registry.NetworkServiceEntry, opts ...grpc.CallOption) (*registry.NetworkServiceEntry, error) {
	return in, nil
}

func (t *echoClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, _ ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	ch := make(chan *registry.NetworkServiceEntry)
	go func() {
		ch <- &registry.NetworkServiceEntry{
			NetworkService: in.NetworkService,
		}
		close(ch)
	}()
	return streamchannel.NewNetworkServiceFindClient(ctx, ch), nil
}

func (t *echoClient) Unregister(_ context.Context, _ *registry.NetworkServiceEntry, _ ...grpc.CallOption) (*empty.Empty, error) {
	return new(empty.Empty), nil
}

func TestNetworkServiceClientToServer_Find(t *testing.T) {
	expected := &registry.NetworkService{
		Name: "echo",
	}
	client := &echoClient{}
	ch := make(chan *registry.NetworkServiceEntry, 1)
	ss := streamchannel.NewNetworkServiceFindServer(context.Background(), ch)
	s := adapters.RegistryClientToServer(client)
	err := s.Find(&registry.NetworkServiceQuery{
		NetworkService: expected,
	}, ss)
	require.Error(t, err)
	require.Equal(t, expected, (<-ch).NetworkService)
}

func TestNetworkServiceClientToServer_Register(t *testing.T) {
	expected := &registry.NetworkService{
		Name: "echo",
	}
	client := &echoClient{}
	s := adapters.RegistryClientToServer(client)
	actual, err := s.Register(context.Background(), &registry.NetworkServiceEntry{
		NetworkService: expected,
	})
	require.NoError(t, err)
	require.Equal(t, expected, actual.NetworkService)
}

func TestNetworkServiceClientToServer_Unregister(t *testing.T) {
	expected := &registry.NetworkService{
		Name: "echo",
	}
	client := &echoClient{}
	s := adapters.RegistryClientToServer(client)
	_, err := s.Unregister(context.Background(), &registry.NetworkServiceEntry{
		NetworkService: expected,
	})
	require.NoError(t, err)
}
