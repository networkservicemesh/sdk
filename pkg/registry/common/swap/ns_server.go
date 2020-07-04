package swap

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomainurl"
)

type swapNSServer struct {
	currentDomain string
}

func (s *swapNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	ns.Name = interdomainurl.Join(ns.Name, s.currentDomain)
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
}

type swapNSFindServer struct {
	registry.NetworkServiceRegistry_FindServer
	domain string
}

func (s *swapNSFindServer) Send(ns *registry.NetworkService) error {
	if !interdomainurl.HasIdentifier(ns.Name) {
		ns.Name = interdomainurl.Join(ns.Name, s.domain)
	}
	return s.NetworkServiceRegistry_FindServer.Send(ns)
}

func (s *swapNSServer) Find(q *registry.NetworkServiceQuery, srv registry.NetworkServiceRegistry_FindServer) error {
	if interdomainurl.HasIdentifier(q.NetworkService.Name) {
		srv = &swapNSFindServer{NetworkServiceRegistry_FindServer: srv, domain: interdomainurl.Domain(q.NetworkService.Name)}
	}
	q.NetworkService.Name = interdomainurl.Name(q.NetworkService.Name)
	return next.NetworkServiceRegistryServer(srv.Context()).Find(q, srv)
}

func (s *swapNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	ns.Name = interdomainurl.Join(ns.Name, s.currentDomain)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}

func NewNetworkServiceServer(domain string) registry.NetworkServiceRegistryServer {
	return &swapNSServer{currentDomain: domain}
}
