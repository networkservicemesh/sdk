package limit_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/limit"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkclose"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type myConnection struct {
	closed atomic.Bool
	grpc.ClientConnInterface
}

func (cc *myConnection) Close() error {
	cc.closed.Store(true)
	return nil
}

func Test_DialLimitShouldCalled_OnLimitReached_Request(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var cc = new(myConnection)
	var myChain = chain.NewNetworkServiceClient(
		metadata.NewClient(),
		clientconn.NewClient(cc),
		limit.NewClient(limit.WithDialLimit(time.Second/5)),
		checkrequest.NewClient(t, func(t *testing.T, nsr *networkservice.NetworkServiceRequest) {
			time.Sleep(time.Second / 4)
		}),
	)

	_, _ = myChain.Request(context.Background(), &networkservice.NetworkServiceRequest{})

	require.Eventually(t, func() bool {
		return cc.closed.Load()
	}, time.Second/2, time.Millisecond*75)
}

func Test_DialLimitShouldCalled_OnLimitReached_Close(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var cc = new(myConnection)
	var myChain = chain.NewNetworkServiceClient(
		metadata.NewClient(),
		clientconn.NewClient(cc),
		limit.NewClient(limit.WithDialLimit(time.Second/5)),
		checkclose.NewClient(t, func(t *testing.T, nsr *networkservice.Connection) {
			time.Sleep(time.Second / 4)
		}),
	)

	_, _ = myChain.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	_, _ = myChain.Close(context.Background(), &networkservice.Connection{})

	require.Eventually(t, func() bool {
		return cc.closed.Load()
	}, time.Second/2, time.Millisecond*75)
}

func Test_DialLimitShouldNotBeCalled_OnSuccesRequest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var cc = new(myConnection)
	var myChain = chain.NewNetworkServiceClient(
		metadata.NewClient(),
		clientconn.NewClient(cc),
		limit.NewClient(limit.WithDialLimit(time.Second/5)),
	)

	_, _ = myChain.Request(context.Background(), &networkservice.NetworkServiceRequest{})

	require.Never(t, func() bool {
		return cc.closed.Load()
	}, time.Second/2, time.Millisecond*75)
}

func Test_DialLimitShouldNotBeCalled_OnSuccessClose(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var cc = new(myConnection)
	var myChain = chain.NewNetworkServiceClient(
		metadata.NewClient(),
		clientconn.NewClient(cc),
		limit.NewClient(limit.WithDialLimit(time.Second/5)),
	)

	_, _ = myChain.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	_, _ = myChain.Close(context.Background(), &networkservice.Connection{})

	require.Never(t, func() bool {
		return cc.closed.Load()
	}, time.Second/2, time.Millisecond*75)
}
