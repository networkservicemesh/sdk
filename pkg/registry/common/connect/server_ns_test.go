package connect_test

import (
	"context"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	"github.com/networkservicemesh/sdk/pkg/registry/common/swap"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"testing"
)

func localNSRegistryServer(proxyDomain string, resolver dnsresolve.Resolver) registry.NetworkServiceRegistryServer {
	return next.NewNetworkServiceRegistryServer(
		memory.NewNetworkServiceRegistryServer(),
		dnsresolve.NewNetworkServiceRegistryServer(
			dnsresolve.WithProxyDomain(proxyDomain),
			dnsresolve.WithResolver(resolver)),
		connect.NewNetworkServiceRegistryServer(),
	)
}

func proxyNSRegistryServer(domain, floatingDomain string, resolver dnsresolve.Resolver) registry.NetworkServiceRegistryServer {
	return next.NewNetworkServiceRegistryServer(
		dnsresolve.NewNetworkServiceRegistryServer(
			dnsresolve.WithProxyDomain(floatingDomain),
			dnsresolve.WithResolver(resolver)),
		swap.NewNetworkServiceServer(domain),
		connect.NewNetworkServiceRegistryServer(),
	)
}

/*
TestInterdomainNetworkServiceRegistry covers the next scenario:
	1. local registry from domain2 has entry "ns-1"
	2. nsmgr from domain1 call find with query "ns-1@domain2"
	3. local registry proxies query to proxy registry
	4. proxy registry proxies query to local registry from domain2

Expected: nsmgr found ns


domain1                                      domain2
 ___________________________________         ___________________
|                                   | Find  |                   |
| local registry --> proxy registry | ----> | local registry    |
|                                   |       |                   |
____________________________________         ___________________
*/
func TestInterdomainNetworkServiceRegistry(t *testing.T) {
	defer goleak.VerifyNone(t)
	tool := newInterdomainTestingTool(t)
	defer tool.cleanup()

	const localRegistryDomain = "domain1.local.registry"
	const proxyRegistryDomain = "domain1.proxy.registry"
	const remoteRegistryDomain = "domain2.local.registry"

	tool.startNetworkServiceRegistryServerAsync(localRegistryDomain, localNSRegistryServer(proxyRegistryDomain, tool))
	tool.startNetworkServiceRegistryServerAsync(proxyRegistryDomain, proxyNSRegistryServer(proxyRegistryDomain, "", tool))

	remoteMem := memory.NewNetworkServiceRegistryServer()
	_, err := remoteMem.Register(context.Background(), &registry.NetworkService{Name: "ns-1"})
	require.Nil(t, err)

	tool.startNetworkServiceRegistryServerAsync(remoteRegistryDomain, remoteMem)

	client := registry.NewNetworkServiceRegistryClient(tool.dialDomain(localRegistryDomain))

	stream, err := client.Find(context.Background(), &registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: "ns-1@" + remoteRegistryDomain,
		},
	})

	require.Nil(t, err)

	list := registry.ReadNetworkServiceList(stream)

	require.Len(t, list, 1)
	require.Equal(t, "ns-1@"+remoteRegistryDomain, list[0].Name)
}

/*
TestInterdomainFloatingNetworkServiceRegistry covers the next scenario:
	1. local registry from domain3 registers entry "ns-1"
	2. proxy registry from domain3 proxies entry "ns-1" to floating registry
	3. nsmgr from domain1 call find with query "ns-1@domain3"
	4. local registry from domain1 proxies query to proxy registry from domain1
	5. proxy registry from domain1 proxies query to floating registry

Expected: nsmgr found ns

domain1	                                        domain2                            domain3
 ___________________________________            ___________________                ___________________________________
|                                   | 2. Find  |                    | 1. Register |                                   |
| local registry --> proxy registry | -------> | floating registry  | <---------  | proxy registry <-- local registry |
|                                   |          |                    |             |	                                  |
____________________________________            ___________________                ___________________________________
*/

func TestInterdomainFloatingNetworkServiceRegistry(t *testing.T) {
	defer goleak.VerifyNone(t)
	tool := newInterdomainTestingTool(t)
	defer tool.cleanup()

	const localRegistryDomain = "domain1.local.registry"
	const proxyRegistryDomain = "domain1.proxy.registry"
	const remoteRegistryDomain = "domain3.local.registry"
	const remoteProxyRegistryDomain = "domain3.proxy.registry"
	const floatingRegistryDomain = "domain2.floating.registry"

	fMem := memory.NewNetworkServiceRegistryServer()

	tool.startNetworkServiceRegistryServerAsync(localRegistryDomain, localNSRegistryServer(proxyRegistryDomain, tool))
	tool.startNetworkServiceRegistryServerAsync(proxyRegistryDomain, proxyNSRegistryServer(proxyRegistryDomain, floatingRegistryDomain, tool))
	tool.startNetworkServiceRegistryServerAsync(floatingRegistryDomain, fMem)
	tool.startNetworkServiceRegistryServerAsync(remoteRegistryDomain, localNSRegistryServer(remoteProxyRegistryDomain, tool))
	tool.startNetworkServiceRegistryServerAsync(remoteProxyRegistryDomain, proxyNSRegistryServer(remoteProxyRegistryDomain, floatingRegistryDomain, tool))

	domain2Client := registry.NewNetworkServiceRegistryClient(tool.dialDomain(remoteRegistryDomain))
	_, err := domain2Client.Register(context.Background(), &registry.NetworkService{
		Name: "ns-1",
	})
	require.Nil(t, err)

	domain1Client := registry.NewNetworkServiceRegistryClient(tool.dialDomain(localRegistryDomain))

	stream, err := domain1Client.Find(context.Background(), &registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: "ns-1@" + remoteProxyRegistryDomain,
		},
	})

	require.Nil(t, err)

	list := registry.ReadNetworkServiceList(stream)

	require.Len(t, list, 1)
	require.Equal(t, "ns-1@"+remoteProxyRegistryDomain, list[0].Name)
}
