// Copyright (c) 2024 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package limit_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/registry/common/limit"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checkcontext"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/metadata"
)

type myConnection struct {
	closed atomic.Bool
	grpc.ClientConnInterface
}

func (cc *myConnection) Close() error {
	cc.closed.Store(true)
	return nil
}

func Test_DialLimitShouldCalled_OnLimitReached(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var cc = new(myConnection)
	var myChain = chain.NewNetworkServiceRegistryClient(
		metadata.NewNetworkServiceClient(),
		clientconn.NewNetworkServiceRegistryClient(),
		checkcontext.NewNSClient(t, func(t *testing.T, ctx context.Context) {
			clientconn.Store(ctx, cc)
		}),
		limit.NewNetworkServiceRegistryClient(limit.WithDialLimit(time.Second/5)),
		checkcontext.NewNSClient(t, func(t *testing.T, ctx context.Context) {
			time.Sleep(time.Second / 5)
		}),
	)

	_, _ = myChain.Register(context.Background(), &registry.NetworkService{Name: t.Name()})

	require.Eventually(t, func() bool {
		return cc.closed.Load()
	}, time.Second/2, time.Millisecond*75)

	cc.closed.Store(false)

	_, _ = myChain.Find(context.Background(), &registry.NetworkServiceQuery{NetworkService: &registry.NetworkService{Name: t.Name()}})

	require.Eventually(t, func() bool {
		return cc.closed.Load()
	}, time.Second/2, time.Millisecond*75)

	cc.closed.Store(false)

	_, _ = myChain.Unregister(context.Background(), &registry.NetworkService{Name: t.Name()})

	require.Eventually(t, func() bool {
		return cc.closed.Load()
	}, time.Second/2, time.Millisecond*75)
}

func Test_DialLimitShouldNotBeCalled_OnSuccess(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var cc = new(myConnection)
	var myChain = chain.NewNetworkServiceRegistryClient(
		metadata.NewNetworkServiceClient(),
		clientconn.NewNetworkServiceRegistryClient(),
		checkcontext.NewNSClient(t, func(t *testing.T, ctx context.Context) {
			clientconn.Store(ctx, cc)
		}),
		limit.NewNetworkServiceRegistryClient(limit.WithDialLimit(time.Second/5)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	_, _ = myChain.Register(ctx, &registry.NetworkService{Name: t.Name()})
	cancel()

	require.Never(t, func() bool {
		return cc.closed.Load()
	}, time.Second/2, time.Millisecond*75)

	ctx, cancel = context.WithCancel(context.Background())
	_, _ = myChain.Find(ctx, &registry.NetworkServiceQuery{NetworkService: &registry.NetworkService{Name: t.Name()}})
	cancel()

	require.Never(t, func() bool {
		return cc.closed.Load()
	}, time.Second/2, time.Millisecond*75)

	ctx, cancel = context.WithCancel(context.Background())
	_, _ = myChain.Unregister(ctx, &registry.NetworkService{Name: t.Name()})
	cancel()

	require.Never(t, func() bool {
		return cc.closed.Load()
	}, time.Second/2, time.Millisecond*75)
}
