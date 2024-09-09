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

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
)

// This test reproduces the situation when refresh changes the eventFactory context.
func TestContextValues_Client(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	checkCtxCl := &checkContextClient{t: t}
	eventFactoryCl := &eventFactoryClient{}
	client := chain.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		checkCtxCl,
		eventFactoryCl,
		&failedNSEClient{},
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
	nse, err := client.Register(ctx, nse.Clone())
	require.NotNil(t, t, nse)
	require.NoError(t, err)

	// Change context value before refresh
	ctx = context.WithValue(ctx, contextKey{}, "value_2")

	// Call refresh that will fail
	nse.Url = failedNSEURLClient
	checkCtxCl.setExpectedValue("value_2")
	_, err = client.Register(ctx, nse.Clone())
	require.Error(t, err)

	// Call refresh from eventFactory. We are expecting the previous value in the context
	checkCtxCl.setExpectedValue("value_1")
	eventFactoryCl.callRefresh()

	// Call refresh that will successful
	nse.Url = ""
	checkCtxCl.setExpectedValue("value_2")
	nse, err = client.Register(ctx, nse.Clone())
	require.NotNil(t, t, nse)
	require.NoError(t, err)

	// Call refresh from eventFactory. We are expecting updated value in the context
	eventFactoryCl.callRefresh()
}

// This test reproduces the situation when Unregister and Register were called at the same time.
func TestRefreshDuringUnregister_Client(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	checkCtxCl := &checkContextClient{t: t}
	eventFactoryCl := &eventFactoryClient{}
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
	nse, err := client.Register(ctx, nse.Clone())
	require.NotNil(t, t, nse)
	require.NoError(t, err)

	// Change context value before refresh
	ctx = context.WithValue(ctx, contextKey{}, "value_2")
	checkCtxCl.setExpectedValue("value_2")

	// Call Unregister from eventFactory
	eventFactoryCl.callUnregister()

	// Call refresh (should be called at the same time as Unregister)
	nse, err = client.Register(ctx, nse.Clone())
	require.NotNil(t, t, nse)
	require.NoError(t, err)

	// Call refresh from eventFactory. We are expecting updated value in the context
	eventFactoryCl.callRefresh()
}

// This test checks if the timeout for the Register/Unregister called from the event factory is correct.
func TestContextTimeout_Client(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add clockMock to the context
	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	ctx, cancel = context.WithDeadline(ctx, clockMock.Now().Add(time.Second*3))
	defer cancel()

	eventFactoryCl := &eventFactoryClient{}
	client := chain.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		eventFactoryCl,
		&delayedNSEClient{t: t, clock: clockMock},
	)

	// Do Register
	nse := &registry.NetworkServiceEndpoint{
		Name: "1",
	}
	nse, err := client.Register(ctx, nse.Clone())
	require.NotNil(t, t, nse)
	require.NoError(t, err)

	// Check eventFactory Refresh. We are expecting the same timeout as for register
	eventFactoryCl.callRefresh()

	// Check eventFactory Unregister. We are expecting the same timeout as for register
	eventFactoryCl.callUnregister()
}

type eventFactoryClient struct {
	registry.NetworkServiceEndpointRegistryClient
	ctx context.Context
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
	eventFactory.Unregister()
}

func (e *eventFactoryClient) callRefresh() {
	eventFactory := begin.FromContext(e.ctx)
	<-eventFactory.Register()
}

type contextKey struct{}

type checkContextClient struct {
	registry.NetworkServiceEndpointRegistryClient
	t             *testing.T
	expectedValue string
}

func (c *checkContextClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	require.Equal(c.t, c.expectedValue, ctx.Value(contextKey{}))
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (c *checkContextClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

func (c *checkContextClient) setExpectedValue(value string) {
	c.expectedValue = value
}

const failedNSEURLClient = "failedNSE"

type failedNSEClient struct {
	registry.NetworkServiceEndpointRegistryClient
}

func (f *failedNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	if in.GetUrl() == failedNSEURLClient {
		return nil, errors.New("failed")
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (f *failedNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	if in.GetUrl() == failedNSEURLClient {
		return nil, errors.New("failed")
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

type delayedNSEClient struct {
	registry.NetworkServiceEndpointRegistryClient
	t              *testing.T
	clock          *clockmock.Mock
	initialTimeout time.Duration
}

func (d *delayedNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	deadline, _ := ctx.Deadline()
	clockTime := clock.FromContext(ctx)
	timeout := clockTime.Until(deadline)

	// Check that context timeout is greater than 0
	require.Greater(d.t, timeout, time.Duration(0))

	// For the first request
	if d.initialTimeout == 0 {
		d.initialTimeout = timeout
	}
	// All requests timeout must be equal the first
	require.Equal(d.t, d.initialTimeout, timeout)

	// Add delay
	d.clock.Add(timeout / 2)
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (d *delayedNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	require.Greater(d.t, d.initialTimeout, time.Duration(0))

	deadline, _ := ctx.Deadline()
	clockTime := clock.FromContext(ctx)

	require.Equal(d.t, d.initialTimeout, clockTime.Until(deadline))
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}
