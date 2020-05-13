package memory_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/stretchr/testify/require"
)

func TestMemoryNetworkServeRegistry_RegisterNSE(t *testing.T) {
	m := memory.NewMemory()
	nsm := &registry.NetworkServiceManager{
		Name: "nsm-1",
	}
	m.NetworkServiceManagers().Put(nsm)
	nse := &registry.NetworkServiceEndpoint{
		Name:                      "nse-1",
		NetworkServiceName:        "ns-1",
		NetworkServiceManagerName: "nsm-1",
	}
	ns := &registry.NetworkService{
		Name: "ns-1",
	}
	m.NetworkServices().Put(ns)
	s := next.NewNetworkServiceRegistryServer(memory.NewNetworkServiceRegistryServer("nsm-1", m))
	resp, err := s.RegisterNSE(context.Background(), nil)
	require.Nil(t, resp)
	require.NotNil(t, err)
	resp, err = s.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkService:         ns,
		NetworkServiceEndpoint: nse,
	})
	require.Nil(t, err)
	require.Equal(t, nsm, resp.NetworkServiceManager)
	require.Equal(t, nsm.Name, resp.NetworkServiceEndpoint.NetworkServiceManagerName)
	require.NotNil(t, m.NetworkServiceEndpoints().Get(nse.Name))
	require.NotNil(t, m.NetworkServices().Get(ns.Name))
}

func TestMemoryNetworkServeRegistry_RemoveNSE(t *testing.T) {
	m := memory.NewMemory()
	nse := &registry.NetworkServiceEndpoint{
		Name:                      "nse-1",
		NetworkServiceName:        "ns-1",
		NetworkServiceManagerName: "nsm-1",
	}
	m.NetworkServiceEndpoints().Put(nse)
	s := next.NewNetworkServiceRegistryServer(memory.NewNetworkServiceRegistryServer("nsm-1", m))
	s.RemoveNSE(context.Background(), &registry.RemoveNSERequest{NetworkServiceEndpointName: "nse-1"})
}
