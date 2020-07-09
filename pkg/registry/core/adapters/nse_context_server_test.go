package adapters_test

import (
	"context"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"testing"
)

// NewServer - returns NetworkServiceRegistryServer that checks the context passed in from the previous Server in the chain
//             t - *testing.T used for the check
//             check - function that checks the context.Context
func NewNSEServer(t *testing.T, check func(*testing.T, context.Context)) registry.NetworkServiceEndpointRegistryServer {
	return &checkContextNSEServer{
		T:     t,
		check: check,
	}
}

type checkContextNSEServer struct {
	*testing.T
	check func(*testing.T, context.Context)
}

func (s *checkContextNSEServer) Register(ctx context.Context, service *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	s.check(s.T, ctx)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, service)
}

func (s *checkContextNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	s.check(s.T, server.Context())
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *checkContextNSEServer) Unregister(ctx context.Context, service *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	s.check(s.T, ctx)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, service)
}

type writeNSEClient struct {
}

func (s *writeNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	ctx = context.WithValue(ctx, "PassingContext", true)
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (s *writeNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	ctx = context.WithValue(ctx, "PassingContext", true)
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
}

func (s *writeNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	ctx = context.WithValue(ctx, "PassingContext", true)
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

func TestNSEPassingContext(t *testing.T) {
	n := next.NewNetworkServiceEndpointRegistryServer(adapters.NetworkServiceEndpointClientToServer(&writeNSEClient{}), NewNSEServer(t, func(t *testing.T, ctx context.Context) {
		if _, ok := ctx.Value("PassingContext").(bool); !ok {
			t.Error("Context ignored")
		}
	}))
	_, _ = n.Register(context.Background(), nil)
	_ = n.Find(nil, streamcontext.NetworkServiceEndpointRegistryFindServer(context.Background(), nil))
	_, _ = n.Unregister(context.Background(), nil)
}
