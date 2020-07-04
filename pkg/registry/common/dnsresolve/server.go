package dnsresolve

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomainurl"
	"net"
)

type dnsNSResolveServer struct {
	resolver      Resolver
	resolveDomain string
}

func (d *dnsNSResolveServer) setResolver(r Resolver) {
	d.resolver = r
}

func (d *dnsNSResolveServer) setDomain(domain string) {
	d.resolveDomain = domain
}

func (d *dnsNSResolveServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	domain := d.resolveDomain
	if domain != "" {
		ctx = connect.WithAddr(ctx, resolveDomain(ctx, domain, d.resolver))
	}
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
}

func (d *dnsNSResolveServer) Find(q *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	if interdomainurl.HasIdentifier(q.NetworkService.Name) {
		domain := d.resolveDomain
		if domain == "" {
			domain = interdomainurl.Domain(q.NetworkService.Name)
		}
		s = streamcontext.NetworkServiceRegistryFindServer(connect.WithAddr(s.Context(), resolveDomain(s.Context(), domain, d.resolver)), s)
	}
	return next.NetworkServiceRegistryServer(s.Context()).Find(q, s)
}

func (d *dnsNSResolveServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	domain := d.resolveDomain
	if domain != "" {
		ctx = connect.WithAddr(ctx, resolveDomain(ctx, domain, d.resolver))
	}
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}

func NewNetworkServiceRegistryServer(options ...Option) registry.NetworkServiceRegistryServer {
	r := &dnsNSResolveServer{
		resolver: net.DefaultResolver,
	}

	for _, o := range options {
		o.apply(r)
	}

	return r
}

var _ Resolver = (*net.Resolver)(nil)
