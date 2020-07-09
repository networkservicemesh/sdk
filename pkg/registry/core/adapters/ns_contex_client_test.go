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
func NewNSClient(t *testing.T, check func(*testing.T, context.Context)) registry.NetworkServiceRegistryClient {
	return &checkContextNSClient{
		T:     t,
		check: check,
	}
}

type checkContextNSClient struct {
	*testing.T
	check func(*testing.T, context.Context)
}

func (c *checkContextNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	c.check(c.T, ctx)
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
}

func (c *checkContextNSClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	c.check(c.T, ctx)
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, in, opts...)
}

func (c *checkContextNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	c.check(c.T, ctx)
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
}

type writeNSServer struct {
}

func (w *writeNSServer) Register(ctx context.Context, service *registry.NetworkService) (*registry.NetworkService, error) {
	ctx = context.WithValue(ctx, "PassingContext", true)
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, service)
}

func (w *writeNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	ctx := context.WithValue(server.Context(), "PassingContext", true)
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, streamcontext.NetworkServiceRegistryFindServer(ctx, server))
}

func (w *writeNSServer) Unregister(ctx context.Context, service *registry.NetworkService) (*empty.Empty, error) {
	ctx = context.WithValue(ctx, "PassingContext", true)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, service)
}

func TestNSClientPassingContext(t *testing.T) {
	n := next.NewNetworkServiceRegistryClient(adapters.NetworkServiceServerToClient(&writeNSServer{}), NewNSClient(t, func(t *testing.T, ctx context.Context) {
		if _, ok := ctx.Value("PassingContext").(bool); !ok {
			t.Error("Context ignored")
		}
	}))
	_, _ = n.Register(context.Background(), nil)
	_, _ = n.Find(context.Background(), nil)
	_, _ = n.Unregister(context.Background(), nil)
}
