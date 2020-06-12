package adapters_test

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	streamchannel "github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"testing"
)

type echoClient struct{}

func (t *echoClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	return in, nil
}

func (t *echoClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, _ ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	ch := make(chan *registry.NetworkService)
	go func() {
		ch <- in.NetworkService
		close(ch)
	}()
	return streamchannel.NewNetworkServiceFindClient(ctx, ch), nil
}

func (t *echoClient) Unregister(_ context.Context, _ *registry.NetworkService, _ ...grpc.CallOption) (*empty.Empty, error) {
	return new(empty.Empty), nil
}

func TestNetworkServiceClientToServer_Register(t *testing.T) {
	expected := &registry.NetworkService{
		Name: "echo",
	}
	client := &echoClient{}
	s := adapters.NetworkServiceClientToServer(client)
	actual, err := s.Register(context.Background(), expected)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestNetworkServiceClientToServer_Unregister(t *testing.T) {
	expected := &registry.NetworkService{
		Name: "echo",
	}
	client := &echoClient{}
	s := adapters.NetworkServiceClientToServer(client)
	_, err := s.Unregister(context.Background(), expected)
	require.NoError(t, err)
}
