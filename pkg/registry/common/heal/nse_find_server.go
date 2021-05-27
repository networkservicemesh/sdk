package heal

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type healNSEFindServer struct {
	ctx    context.Context
	active bool

	*healNSEServer
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (s *healNSEFindServer) Send(nse *registry.NetworkServiceEndpoint) error {
	panic("implement me")
}

func (s *healNSEFindServer) Context() context.Context {
	return s.ctx
}
