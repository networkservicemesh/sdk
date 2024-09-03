// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
)

// This test reproduces the situation when refresh changes the eventFactory context
//nolint:dupl
func TestContextValues_Client(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	checkCtxCl := &checkContextClient{t: t}
	eventFactoryCl := &eventFactoryClient{}
	client := chain.NewNetworkServiceClient(
		begin.NewClient(),
		checkCtxCl,
		eventFactoryCl,
		&failedNSEClient{},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set any value to context
	ctx = context.WithValue(ctx, contextKey{}, "value_1")
	checkCtxCl.setExpectedValue("value_1")

	// Do Request with this context
	request := testRequest("1")
	conn, err := client.Request(ctx, request.Clone())
	require.NotNil(t, t, conn)
	require.NoError(t, err)

	// Change context value before refresh Request
	ctx = context.WithValue(ctx, contextKey{}, "value_2")

	// Call refresh that will fail
	request.Connection = conn.Clone()
	request.Connection.NetworkServiceEndpointName = failedNSENameClient
	checkCtxCl.setExpectedValue("value_2")
	_, err = client.Request(ctx, request.Clone())
	require.Error(t, err)

	// Call refresh from eventFactory. We are expecting the previous value in the context
	checkCtxCl.setExpectedValue("value_1")
	eventFactoryCl.callRefresh()

	// Call refresh that will successful
	request.Connection.NetworkServiceEndpointName = ""
	checkCtxCl.setExpectedValue("value_2")
	conn, err = client.Request(ctx, request.Clone())
	require.NotNil(t, t, conn)
	require.NoError(t, err)

	// Call refresh from eventFactory. We are expecting updated value in the context
	eventFactoryCl.callRefresh()
}

// This test reproduces the situation when Close and Request were called at the same time
// nolint:dupl
func TestRefreshDuringClose_Client(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	checkCtxCl := &checkContextClient{t: t}
	eventFactoryCl := &eventFactoryClient{}
	client := chain.NewNetworkServiceClient(
		begin.NewClient(),
		checkCtxCl,
		eventFactoryCl,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set any value to context
	ctx = context.WithValue(ctx, contextKey{}, "value_1")
	checkCtxCl.setExpectedValue("value_1")

	// Do Request with this context
	request := testRequest("1")
	conn, err := client.Request(ctx, request.Clone())
	require.NotNil(t, t, conn)
	require.NoError(t, err)

	// Change context value before refresh Request
	ctx = context.WithValue(ctx, contextKey{}, "value_2")
	checkCtxCl.setExpectedValue("value_2")
	request.Connection = conn.Clone()

	// Call Close from eventFactory
	eventFactoryCl.callClose()

	// Call refresh  (should be called at the same time as Close)
	conn, err = client.Request(ctx, request.Clone())
	require.NotNil(t, t, conn)
	require.NoError(t, err)

	// Call refresh from eventFactory. We are expecting updated value in the context
	eventFactoryCl.callRefresh()
}

// This test checks if the timeout for the Request/Close called from the event factory is correct.
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
	client := chain.NewNetworkServiceClient(
		begin.NewClient(),
		eventFactoryCl,
		&delayedNSEClient{t: t, clock: clockMock},
	)

	// Do Request
	request := testRequest("1")
	conn, err := client.Request(ctx, request.Clone())
	require.NotNil(t, t, conn)
	require.NoError(t, err)

	// Check eventFactory Refresh. We are expecting the same timeout as for request
	eventFactoryCl.callRefresh()

	// Check eventFactory Close. We are expecting the same timeout as for request
	eventFactoryCl.callClose()
}

// This test checks if the eventFactory reselect option cancels Close ctx correctly.
func TestContextCancellationOnReselect(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	var closeCall atomic.Bool
	ch := make(chan struct{}, 1)

	eventFactoryCl := &eventFactoryClient{}
	counter := &count.Client{}
	client := chain.NewNetworkServiceClient(
		begin.NewClient(),
		eventFactoryCl,
		checkcontext.NewClient(t, func(t *testing.T, reqCtx context.Context) {
			if !closeCall.Load() {
				<-ch
			} else {
				closeCall.Store(false)
				go func() {
					<-reqCtx.Done()
					ch <- struct{}{}
				}()
			}
		}),
		counter,
	)

	// Do Request
	// Write to ch for the first request
	ch <- struct{}{}
	request := testRequest("1")
	conn, err := client.Request(ctx, request.Clone())
	require.NotNil(t, t, conn)
	require.NoError(t, err)

	// Call Reselect
	// It will call Close first
	closeCall.Store(true)
	eventFactoryCl.callReselect()

	// Waiting for the re-request
	require.Eventually(t, func() bool {
		return counter.Requests() == 2
	}, time.Second, time.Millisecond*100)
}

type eventFactoryClient struct {
	ctx context.Context
}

func (s *eventFactoryClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	s.ctx = ctx
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (s *eventFactoryClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	// Wait to be sure that rerequest was called
	time.Sleep(time.Millisecond * 100)
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func (s *eventFactoryClient) callClose() {
	eventFactory := begin.FromContext(s.ctx)
	eventFactory.Close()
}

func (s *eventFactoryClient) callRefresh() {
	eventFactory := begin.FromContext(s.ctx)
	<-eventFactory.Request()
}

func (s *eventFactoryClient) callReselect() {
	eventFactory := begin.FromContext(s.ctx)
	eventFactory.Request(begin.WithReselect())
}

type contextKey struct{}

type checkContextClient struct {
	t             *testing.T
	expectedValue string
}

func (c *checkContextClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	require.Equal(c.t, c.expectedValue, ctx.Value(contextKey{}))
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *checkContextClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func (c *checkContextClient) setExpectedValue(value string) {
	c.expectedValue = value
}

const failedNSENameClient = "failedNSE"

type failedNSEClient struct{}

func (f *failedNSEClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if request.GetConnection().GetNetworkServiceEndpointName() == failedNSENameClient {
		return nil, errors.New("failed")
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (f *failedNSEClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if conn.GetNetworkServiceEndpointName() == failedNSENameClient {
		return nil, errors.New("failed")
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}

type delayedNSEClient struct {
	t              *testing.T
	clock          *clockmock.Mock
	initialTimeout time.Duration
}

func (d *delayedNSEClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
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
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (d *delayedNSEClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	require.Greater(d.t, d.initialTimeout, time.Duration(0))

	deadline, _ := ctx.Deadline()
	clockTime := clock.FromContext(ctx)

	require.Equal(d.t, d.initialTimeout, clockTime.Until(deadline))

	return next.Client(ctx).Close(ctx, conn, opts...)
}
