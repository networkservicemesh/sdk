package heal

import (
	"context"
	"math/rand"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type healNSEClient struct {
	id       int
	ctx      context.Context
	err      error
	nseInfos nseInfoMap
	lock     sync.RWMutex
	once     sync.Once
}

type nseInfo struct {
	nse    *registry.NetworkServiceEndpoint
	active bool
	lock   sync.Mutex
}

func NewNetworkServiceEndpointRegistryClient(ctx context.Context) registry.NetworkServiceEndpointRegistryClient {
	return &healNSEClient{
		id:  rand.Int(),
		ctx: ctx,
	}
}

func (c *healNSEClient) init(ctx context.Context, opts ...grpc.CallOption) error {
	c.once.Do(func() {
		var stream registry.NetworkServiceEndpointRegistry_FindClient
		if stream, c.err = next.NetworkServiceEndpointRegistryClient(ctx).Find(c.ctx, &registry.NetworkServiceEndpointQuery{
			NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
			Watch:                  true,
		}, opts...); c.err != nil {
			return
		}

		go c.monitorUpdates(stream, requestNSERestore(ctx))
	})

	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.err
}

func (c *healNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	if err := c.init(ctx, opts...); err != nil {
		return nil, err
	}

	info, loaded := c.nseInfos.LoadOrStore(nse.Name, &nseInfo{
		nse: nse.Clone(),
	})
	nse.ExpirationTime = proto.Clone(nse.ExpirationTime).(*timestamppb.Timestamp)

	reg, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
	if err != nil {
		if !loaded {
			c.nseInfos.Delete(nse.Name)
		}
		return nil, err
	}

	info.lock.Lock()
	defer info.lock.Unlock()

	info.nse.ExpirationTime = proto.Clone(reg.ExpirationTime).(*timestamppb.Timestamp)

	return reg, nil
}

func (c *healNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *healNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	info, loaded := c.nseInfos.LoadAndDelete(nse.Name)
	if !loaded {
		return new(empty.Empty), nil
	}

	info.lock.Lock()

	info.active = false
	nse.ExpirationTime = proto.Clone(info.nse.ExpirationTime).(*timestamppb.Timestamp)

	info.lock.Unlock()

	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}

func (c *healNSEClient) monitorUpdates(stream registry.NetworkServiceEndpointRegistry_FindClient, requestRestore requestNSERestoreFunc) {
	nse, err := stream.Recv()
	for ; err == nil; nse, err = stream.Recv() {
		if info, ok := c.nseInfos.Load(nse.Name); ok {
			c.processUpdate(nse, info)
		}
	}

	c.lock.Lock()

	c.err = err

	c.lock.Unlock()

	c.nseInfos.Range(func(_ string, info *nseInfo) bool {
		if info.active {
			requestRestore(info.nse)
		}
		return true
	})
}

func (c *healNSEClient) processUpdate(nse *registry.NetworkServiceEndpoint, info *nseInfo) {
	info.lock.Lock()
	defer info.lock.Unlock()

	if nse.ExpirationTime != nil && nse.ExpirationTime.Seconds == -1 {
		// TODO: should we do something on Unregister update?
		return
	}

	info.active = true
	info.nse.ExpirationTime = proto.Clone(nse.ExpirationTime).(*timestamppb.Timestamp)
}
