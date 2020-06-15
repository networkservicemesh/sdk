package memory

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/matchutils"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
)

type networkServiceRegistryServer struct {
	networkServices     NetworkServiceSyncMap
	executor            serialize.Executor
	eventChannels       []chan *registry.NetworkService
	eventChannelSize    int
	eventChannelsLocker sync.Mutex
}

func (n *networkServiceRegistryServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	n.networkServices.Store(ns.Name, ns)
	n.executor.AsyncExec(func() {
		n.eventChannelsLocker.Lock()
		for _, ch := range n.eventChannels {
			ch <- ns
		}
		n.eventChannelsLocker.Unlock()
	})
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
}

func (n *networkServiceRegistryServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	sendAllMatches := func(ns *registry.NetworkService) error {
		var err error
		n.networkServices.Range(func(key string, value *registry.NetworkService) bool {
			if matchutils.MatchNetworkServices(query.NetworkService, value) {
				err = s.Send(value)
				return err == nil
			}
			return true
		})
		return err
	}
	if query.Watch {
		go func() {
			eventCh := make(chan *registry.NetworkService, n.eventChannelSize)
			n.eventChannelsLocker.Lock()
			n.eventChannels = append(n.eventChannels, eventCh)
			n.eventChannelsLocker.Unlock()
			err := sendAllMatches(query.NetworkService)
			if err != nil {
				return
			}
			for {
				select {
				case <-s.Context().Done():
					return
				case event := <-eventCh:
					if matchutils.MatchNetworkServices(query.NetworkService, event) {
						if s.Context().Err() != nil {
							return
						}
						if err := s.Send(event); err != nil {
							return
						}
					}
				}
			}
		}()
	} else {
		if err := sendAllMatches(query.NetworkService); err != nil {
			return err
		}
	}
	return next.NetworkServiceRegistryServer(s.Context()).Find(query, s)
}

func (n *networkServiceRegistryServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	n.networkServices.Delete(ns.Name)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}

func NewNetworkServiceRegistryServer() registry.NetworkServiceRegistryServer {
	return &networkServiceRegistryServer{
		eventChannelSize: 10,
	}
}
