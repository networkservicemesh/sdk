// Copyright (c) 2022 Cisco and/or its affiliates.
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

package begin_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
)

// This test reproduces the situation when Unregister and Register were called at the same time
func TestRefreshDuringUnregister_Client(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	syncChan := make(chan struct{})
	checkCtxCl := &checkContextClient{t: t}
	eventFactoryCl := &eventFactoryClient{ch: syncChan}
	client := chain.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		checkCtxCl,
		eventFactoryCl,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set any value to context
	ctx = context.WithValue(ctx, contextKey{}, "value_1")
	checkCtxCl.setExpectedValue("value_1")

	// Do Register with this context
	nse := &registry.NetworkServiceEndpoint{
		Name: "1",
	}
	conn, err := client.Register(ctx, nse.Clone())
	assert.NotNil(t, t, conn)
	assert.NoError(t, err)

	// Change context value before refresh
	ctx = context.WithValue(ctx, contextKey{}, "value_2")
	checkCtxCl.setExpectedValue("value_2")

	// Call Unregister from eventFactory
	eventFactoryCl.callUnregister()
	<-syncChan

	// Call refresh (should be called at the same time as Unregister)
	conn, err = client.Register(ctx, nse.Clone())
	assert.NotNil(t, t, conn)
	assert.NoError(t, err)

	// Call refresh from eventFactory. We are expecting updated value in the context
	eventFactoryCl.callRefresh()
	<-syncChan
}

type eventFactoryClient struct {
	registry.NetworkServiceEndpointRegistryClient
	ctx context.Context
	ch  chan<- struct{}
}

func (e *eventFactoryClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	e.ctx = ctx
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (e *eventFactoryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	// Wait to be sure that reregister was called
	time.Sleep(time.Millisecond * 100)
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

func (e *eventFactoryClient) callUnregister() {
	eventFactory := begin.FromContext(e.ctx)
	go func() {
		e.ch <- struct{}{}
		eventFactory.Unregister()
	}()
}

func (e *eventFactoryClient) callRefresh() {
	eventFactory := begin.FromContext(e.ctx)
	go func() {
		e.ch <- struct{}{}
		eventFactory.Register()
	}()
}

type contextKey struct{}

type checkContextClient struct {
	registry.NetworkServiceEndpointRegistryClient
	t             *testing.T
	expectedValue string
}

func (c *checkContextClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	assert.Equal(c.t, c.expectedValue, ctx.Value(contextKey{}))
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (c *checkContextClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

func (c *checkContextClient) setExpectedValue(value string) {
	c.expectedValue = value
}
