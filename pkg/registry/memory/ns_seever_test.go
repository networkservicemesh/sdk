package memory_test

import (
	"context"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"testing"
)

func TestNetworkServiceRegistryServer_RegisterAndFind(t *testing.T) {
	defer goleak.VerifyNone(t)
	s := next.NewNetworkServiceRegistryServer(memory.NewNetworkServiceRegistryServer())

	_, err := s.Register(context.Background(), &registry.NetworkService{
		Name: "a",
	})
	require.NoError(t, err)

	_, err = s.Register(context.Background(), &registry.NetworkService{
		Name: "b",
	})
	require.NoError(t, err)

	_, err = s.Register(context.Background(), &registry.NetworkService{
		Name: "c",
	})
	require.NoError(t, err)

	ch := make(chan *registry.NetworkService, 1)
	_ = s.Find(&registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: "a",
		},
	}, streamchannel.NewNetworkServiceFindServer(context.Background(), ch))

	require.Equal(t, &registry.NetworkService{
		Name: "a",
	}, <-ch)

}

func TestNetworkServiceRegistryServer_RegisterAndFindWatch(t *testing.T) {
	defer goleak.VerifyNone(t)
	s := next.NewNetworkServiceRegistryServer(memory.NewNetworkServiceRegistryServer())

	_, err := s.Register(context.Background(), &registry.NetworkService{
		Name: "a",
	})
	require.NoError(t, err)

	_, err = s.Register(context.Background(), &registry.NetworkService{
		Name: "b",
	})
	require.NoError(t, err)

	_, err = s.Register(context.Background(), &registry.NetworkService{
		Name: "c",
	})
	require.NoError(t, err)

	ch := make(chan *registry.NetworkService, 1)
	_ = s.Find(&registry.NetworkServiceQuery{
		Watch: true,
		NetworkService: &registry.NetworkService{
			Name: "a",
		},
	}, streamchannel.NewNetworkServiceFindServer(context.Background(), ch))

	require.Equal(t, &registry.NetworkService{
		Name: "a",
	}, <-ch)

	_, err = s.Register(context.Background(), &registry.NetworkService{
		Name: "a",
	})
	require.NoError(t, err)

	require.Equal(t, &registry.NetworkService{
		Name: "a",
	}, <-ch)

	close(ch)
}
