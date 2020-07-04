package connect

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"google.golang.org/grpc"
	"net"
	"sync"
)

type cacheEntry struct {
	client registry.NetworkServiceRegistryClient
	close  func() error
}

type connectNSServer struct {
	cacheLock sync.Mutex
	cache     map[string]*cacheEntry
}

func (c *connectNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	if addr := Addr(ctx); addr != nil {
		entry, err := c.getClient(addr)
		if err != nil {
			return nil, err
		}
		defer entry.close()
		ns, err = entry.client.Register(ctx, ns)
		if err != nil {
			return nil, err
		}
	}
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
}

func (c *connectNSServer) getClient(addr *net.TCPAddr) (*cacheEntry, error) {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if v, ok := c.cache[addr.String()]; ok {
		return v, nil
	}

	conn, err := grpc.Dial(addr.String(), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	entry := &cacheEntry{
		client: registry.NewNetworkServiceRegistryClient(conn),
		close:  conn.Close,
	}

	c.cache[addr.String()] = entry

	return entry, nil
}

func (c connectNSServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	if addr := Addr(s.Context()); addr != nil {
		entry, err := c.getClient(addr)
		if err != nil {
			return err
		}
		defer entry.close()
		stream, err := entry.client.Find(s.Context(), query)
		if err != nil {
			return err
		}

		for ns := range registry.ReadNetworkServiceChannel(stream) {
			s.Send(ns)
		}
	}
	return next.NetworkServiceRegistryServer(s.Context()).Find(query, s)
}

func (c connectNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	if addr := Addr(ctx); addr != nil {
		entry, err := c.getClient(addr)
		if err != nil {
			return nil, err
		}
		defer entry.close()
		ns, err = entry.client.Register(ctx, ns)
		if err != nil {
			return nil, err
		}
	}
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}

func NewNetworkServiceRegistryServer() registry.NetworkServiceRegistryServer {
	r := &connectNSServer{
		cache: map[string]*cacheEntry{},
	}
	return r
}

var _ registry.NetworkServiceRegistryServer = &connectNSServer{}
