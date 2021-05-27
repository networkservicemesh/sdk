package heal

import (
	"context"
	"math/rand"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type healNSClient struct {
	id      int
	ctx     context.Context
	err     error
	nsInfos nsInfoMap
	lock    sync.RWMutex
	once    sync.Once
}

type nsInfo struct {
	ns     *registry.NetworkService
	active bool
	lock   sync.Mutex
}

func NewNetworkServiceRegistryClient(ctx context.Context) registry.NetworkServiceRegistryClient {
	return &healNSClient{
		id:  rand.Int(),
		ctx: ctx,
	}
}

func (c *healNSClient) init(ctx context.Context, opts ...grpc.CallOption) error {
	c.once.Do(func() {
		var stream registry.NetworkServiceRegistry_FindClient
		if stream, c.err = next.NetworkServiceRegistryClient(ctx).Find(c.ctx, &registry.NetworkServiceQuery{
			NetworkService: new(registry.NetworkService),
			Watch:          true,
		}, opts...); c.err != nil {
			return
		}

		go c.monitorUpdates(stream, requestNSRestore(ctx))
	})

	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.err
}

func (c *healNSClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	if err := c.init(ctx, opts...); err != nil {
		return nil, err
	}

	info, loaded := c.nsInfos.LoadOrStore(ns.Name, &nsInfo{
		ns: ns.Clone(),
	})

	reg, err := next.NetworkServiceRegistryClient(ctx).Register(ctx, ns, opts...)
	if err != nil {
		if !loaded {
			c.nsInfos.Delete(ns.Name)
		}
		return nil, err
	}

	info.lock.Lock()
	defer info.lock.Unlock()

	return reg, nil
}

func (c *healNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *healNSClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	info, loaded := c.nsInfos.LoadAndDelete(ns.Name)
	if !loaded {
		return new(empty.Empty), nil
	}

	info.lock.Lock()

	info.active = false

	info.lock.Unlock()

	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, ns, opts...)
}

func (c *healNSClient) monitorUpdates(stream registry.NetworkServiceRegistry_FindClient, requestRestore requestNSRestoreFunc) {
	ns, err := stream.Recv()
	for ; err == nil; ns, err = stream.Recv() {
		if info, ok := c.nsInfos.Load(ns.Name); ok {
			info.lock.Lock()

			info.active = true

			info.lock.Unlock()
		}
	}

	c.lock.Lock()

	c.err = err

	c.lock.Unlock()

	c.nsInfos.Range(func(_ string, info *nsInfo) bool {
		if info.active {
			requestRestore(info.ns)
		}
		return true
	})
}
