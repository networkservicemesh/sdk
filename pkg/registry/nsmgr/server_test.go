package nsmgr_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/nsmgr"
)

func TestRegisterNSE(t *testing.T) {
	storage := &memory.Storage{}
	mgr := &registry.NetworkServiceManager{
		Name: "nsm-1",
		Url:  "tcp://127.0.0.1:5000",
	}
	storage.NetworkServiceManagers.Store("nsm-1", mgr)
	nseReg := chain.NewNetworkServiceRegistryServer(nsmgr.NewServer(mgr), memory.NewNetworkServiceRegistryServer(mgr.Name, storage))
	regEndpoint, err := nseReg.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name: "my-service",
		},
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name:               "my-endpoint",
			NetworkServiceName: "my-service",
			Url:                "tcp://127.0.0.1:5001",
		},
	})
	require.Nil(t, err)
	require.NotNil(t, regEndpoint)
	require.Equal(t, mgr.Name, regEndpoint.NetworkServiceManager.Name)
}

func TestRegisterNSEStream(t *testing.T) {
	storage := &memory.Storage{}
	mgr := &registry.NetworkServiceManager{
		Name: "nsm-1",
		Url:  "tcp://127.0.0.1:5000",
	}
	storage.NetworkServiceManagers.Store("nsm-1", mgr)
	nseReg := chain.NewNetworkServiceRegistryServer(nsmgr.NewServer(mgr), memory.NewNetworkServiceRegistryServer(mgr.Name, storage))

	client := adapters.NewRegistryServerToClient(nseReg)

	regClient, err := client.BulkRegisterNSE(context.Background())
	require.Nil(t, err)

	err = regClient.Send(&registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name: "my-service",
		},
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name:               "my-endpoint",
			NetworkServiceName: "my-service",
			Url:                "tcp://127.0.0.1:5001",
		},
	})
	require.Nil(t, err)
	var regEndpoint *registry.NSERegistration
	regEndpoint, err = regClient.Recv()
	require.Nil(t, err)
	require.NotNil(t, regEndpoint)
	require.Equal(t, mgr.Name, regEndpoint.NetworkServiceManager.Name)
}
