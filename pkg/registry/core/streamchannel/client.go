package streamchannel

import (
	"context"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func NewNetworkServiceFindClient(ctx context.Context, recvCh <-chan *registry.NetworkServiceEntry) registry.NetworkServiceRegistry_FindClient {
	return &networkServiceRegistryFindClient{
		ctx:    ctx,
		recvCh: recvCh,
	}
}

type networkServiceRegistryFindClient struct {
	grpc.ClientStream
	err    error
	recvCh <-chan *registry.NetworkServiceEntry
	ctx    context.Context
}

func (c *networkServiceRegistryFindClient) Recv() (*registry.NetworkServiceEntry, error) {
	res, ok := <-c.recvCh
	if !ok {
		err := errors.New("recv channel has been closed")
		if c.err == nil {
			return nil, err
		}
		return res, errors.Wrap(c.err, err.Error())
	}
	return res, c.err
}

func (c *networkServiceRegistryFindClient) Context() context.Context {
	return c.ctx
}

var _ registry.NetworkServiceRegistry_FindClient = &networkServiceRegistryFindClient{}
