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

// NewNSServer - returns NetworkServiceRegistryServer that checks the context passed in from the previous NSServer in the chain
//             t - *testing.T used for the check
//             check - function that checks the context.Context
func NewNSServer(t *testing.T, check func(*testing.T, context.Context)) registry.NetworkServiceRegistryServer {
	return &checkContextNSServer{
		T:     t,
		check: check,
	}
}

type checkContextNSServer struct {
	*testing.T
	check func(*testing.T, context.Context)
}

func (c *checkContextNSServer) Register(ctx context.Context, service *registry.NetworkService) (*registry.NetworkService, error) {
	c.check(c.T, ctx)
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, service)
}

func (c *checkContextNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	c.check(c.T, server.Context())
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (c *checkContextNSServer) Unregister(ctx context.Context, service *registry.NetworkService) (*empty.Empty, error) {
	c.check(c.T, ctx)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, service)
}

type writeNSClient struct {
}

func (w *writeNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	ctx = context.WithValue(ctx, "PassingContext", true)
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
}

func (w *writeNSClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	ctx = context.WithValue(ctx, "PassingContext", true)
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, in, opts...)
}

func (w *writeNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	ctx = context.WithValue(ctx, "PassingContext", true)
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
}

func TestNSServerPassingContext(t *testing.T) {
	n := next.NewNetworkServiceRegistryServer(adapters.NetworkServiceClientToServer(&writeNSClient{}), NewNSServer(t, func(t *testing.T, ctx context.Context) {
		if _, ok := ctx.Value("PassingContext").(bool); !ok {
			t.Error("Context ignored")
		}
	}))
	_, _ = n.Register(context.Background(), nil)
	_ = n.Find(nil, streamcontext.NetworkServiceRegistryFindServer(context.Background(), nil))
	_, _ = n.Unregister(context.Background(), nil)
}
