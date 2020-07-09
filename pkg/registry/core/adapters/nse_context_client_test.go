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

// NewNSClient - returns NetworkServiceRegistryClient that checks the context passed in from the previous NSClient in the chain
//             t - *testing.T used for the check
//             check - function that checks the context.Context
func NewNSEClient(t *testing.T, check func(*testing.T, context.Context)) registry.NetworkServiceEndpointRegistryClient {
	return &checkContextNSEClient{
		T:     t,
		check: check,
	}
}

type checkContextNSEClient struct {
	*testing.T
	check func(*testing.T, context.Context)
}

func (c *checkContextNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	c.check(c.T, ctx)
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (c *checkContextNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	c.check(c.T, ctx)
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
}

func (c *checkContextNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	c.check(c.T, ctx)
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

type writeNSEServer struct {
}

func (w *writeNSEServer) Register(ctx context.Context, service *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	ctx = context.WithValue(ctx, "PassingContext", true)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, service)
}

func (w *writeNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	ctx := context.WithValue(server.Context(), "PassingContext", true)
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, server))
}

func (w *writeNSEServer) Unregister(ctx context.Context, service *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	ctx = context.WithValue(ctx, "PassingContext", true)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, service)
}

func TestNSEClientPassingContext(t *testing.T) {
	n := next.NewNetworkServiceEndpointRegistryClient(adapters.NetworkServiceEndpointServerToClient(&writeNSEServer{}), NewNSEClient(t, func(t *testing.T, ctx context.Context) {
		if _, ok := ctx.Value("PassingContext").(bool); !ok {
			t.Error("Context ignored")
		}
	}))
	_, _ = n.Register(context.Background(), nil)
	_, _ = n.Find(context.Background(), nil)
	_, _ = n.Unregister(context.Background(), nil)
}
