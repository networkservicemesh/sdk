package endpointurls

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type endpointURLsServer struct {
	set *Set
}

func (e *endpointURLsServer) Register(ctx context.Context, endpoint *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if u, err := url.Parse(endpoint.Url); err == nil {
		e.set.Store(*u, struct{}{})
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, endpoint)
}

func (e *endpointURLsServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (e *endpointURLsServer) Unregister(ctx context.Context, endpoint *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if u, err := url.Parse(endpoint.Url); err == nil {
		e.set.Delete(*u)
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, endpoint)
}

func NewNetworkServiceEndpointRegistryServer(set *Set) registry.NetworkServiceEndpointRegistryServer {
	return &endpointURLsServer{set: set}
}
