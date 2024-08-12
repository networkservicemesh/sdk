// Copyright (c) 2022-2024 Cisco and/or its affiliates.
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
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

// This test reproduces the situation when refresh changes the eventFactory context
// nolint:dupl
func TestContextValues_Server(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	checkCtxServ := &checkContextServer{t: t}
	eventFactoryServ := &eventFactoryServer{}
	server := chain.NewNetworkServiceServer(
		begin.NewServer(),
		checkCtxServ,
		eventFactoryServ,
		&failedNSEServer{},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set any value to context
	ctx = context.WithValue(ctx, contextKey{}, "value_1")
	checkCtxServ.setExpectedValue("value_1")

	// Do Request with this context
	request := testRequest("1")
	conn, err := server.Request(ctx, request.Clone())
	require.NotNil(t, t, conn)
	require.NoError(t, err)

	// Change context value before refresh Request
	ctx = context.WithValue(ctx, contextKey{}, "value_2")

	// Call refresh that will fail
	request.Connection = conn.Clone()
	request.Connection.NetworkServiceEndpointName = failedNSENameServer
	checkCtxServ.setExpectedValue("value_2")
	_, err = server.Request(ctx, request.Clone())
	require.Error(t, err)

	// Call refresh from eventFactory. We are expecting the previous value in the context
	checkCtxServ.setExpectedValue("value_1")
	eventFactoryServ.callRefresh()

	// Call refresh that will successful
	request.Connection.NetworkServiceEndpointName = ""
	checkCtxServ.setExpectedValue("value_2")
	conn, err = server.Request(ctx, request.Clone())
	require.NotNil(t, t, conn)
	require.NoError(t, err)

	// Call refresh from eventFactory. We are expecting updated value in the context
	eventFactoryServ.callRefresh()
}

// This test reproduces the situation when Close and Request were called at the same time
// nolint:dupl
func TestRefreshDuringClose_Server(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	checkCtxServ := &checkContextServer{t: t}
	eventFactoryServ := &eventFactoryServer{}
	server := chain.NewNetworkServiceServer(
		begin.NewServer(),
		checkCtxServ,
		eventFactoryServ,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set any value to context
	ctx = context.WithValue(ctx, contextKey{}, "value_1")
	checkCtxServ.setExpectedValue("value_1")

	// Do Request with this context
	request := testRequest("1")
	conn, err := server.Request(ctx, request.Clone())
	require.NotNil(t, t, conn)
	require.NoError(t, err)

	// Change context value before refresh Request
	ctx = context.WithValue(ctx, contextKey{}, "value_2")
	checkCtxServ.setExpectedValue("value_2")
	request.Connection = conn.Clone()

	// Call Close from eventFactory
	eventFactoryServ.callClose()

	// Call refresh  (should be called at the same time as Close)
	conn, err = server.Request(ctx, request.Clone())
	require.NotNil(t, t, conn)
	require.NoError(t, err)

	// Call refresh from eventFactory. We are expecting updated value in the context
	eventFactoryServ.callRefresh()
}

// This test checks if the timeout for the Request/Close called from the event factory is correct
func TestContextTimeout_Server(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	contextTimeout := time.Second * 2
	eventFactoryServ := &eventFactoryServer{}
	server := chain.NewNetworkServiceServer(
		begin.NewServer(begin.WithContextTimeout(contextTimeout)),
		eventFactoryServ,
		&delayedNSEServer{t: t, contextTimeout: contextTimeout},
	)

	// Do Request
	request := testRequest("1")
	conn, err := server.Request(ctx, request.Clone())
	require.NotNil(t, t, conn)
	require.NoError(t, err)

	// Check eventFactory Refresh. We are expecting the same timeout as for request
	eventFactoryServ.callRefresh()

	// Check eventFactory Close. We are expecting the same timeout as for request
	eventFactoryServ.callClose()
}

type eventFactoryServer struct {
	ctx context.Context
}

func (e *eventFactoryServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	e.ctx = ctx
	return next.Server(ctx).Request(ctx, request)
}

func (e *eventFactoryServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	// Wait to be sure that rerequest was called
	time.Sleep(time.Millisecond * 100)
	return next.Server(ctx).Close(ctx, conn)
}

func (e *eventFactoryServer) callClose() {
	eventFactory := begin.FromContext(e.ctx)
	eventFactory.Close()
}

func (e *eventFactoryServer) callRefresh() {
	eventFactory := begin.FromContext(e.ctx)
	<-eventFactory.Request()
}

type checkContextServer struct {
	t             *testing.T
	expectedValue string
}

func (c *checkContextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	require.Equal(c.t, c.expectedValue, ctx.Value(contextKey{}))
	return next.Server(ctx).Request(ctx, request)
}

func (c *checkContextServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

func (c *checkContextServer) setExpectedValue(value string) {
	c.expectedValue = value
}

const failedNSENameServer = "failedNSE"

type failedNSEServer struct{}

func (f *failedNSEServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().NetworkServiceEndpointName == failedNSENameServer {
		return nil, errors.New("failed")
	}
	return next.Server(ctx).Request(ctx, request)
}

func (f *failedNSEServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	if conn.NetworkServiceEndpointName == failedNSENameServer {
		return nil, errors.New("failed")
	}
	return next.Server(ctx).Close(ctx, conn)
}

type delayedNSEServer struct {
	t              *testing.T
	initialTimeout time.Duration
	contextTimeout time.Duration
}

func (d *delayedNSEServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
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
	require.Less(d.t, (d.initialTimeout - timeout).Abs(), time.Second)

	// Add delay
	time.Sleep(timeout / 2)
	return next.Server(ctx).Request(ctx, request)
}

func (d *delayedNSEServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	require.Greater(d.t, d.initialTimeout, time.Duration(0))

	deadline, _ := ctx.Deadline()
	timeout := time.Until(deadline)

	require.Less(d.t, (d.contextTimeout - timeout).Abs(), time.Second)

	return next.Server(ctx).Close(ctx, conn)
}
