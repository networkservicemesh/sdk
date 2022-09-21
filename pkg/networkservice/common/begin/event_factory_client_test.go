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

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// This test reproduces the situation when Close and Request were called at the same time
// nolint:dupl
func TestRefreshDuringClose_Client(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	syncChan := make(chan struct{})
	checkCtxCl := &checkContextClient{t: t}
	eventFactoryCl := &eventFactoryClient{ch: syncChan}
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
	assert.NotNil(t, t, conn)
	assert.NoError(t, err)

	// Change context value before refresh Request
	ctx = context.WithValue(ctx, contextKey{}, "value_2")
	checkCtxCl.setExpectedValue("value_2")
	request.Connection = conn.Clone()

	// Call Close from eventFactory
	eventFactoryCl.callClose()
	<-syncChan

	// Call refresh  (should be called at the same time as Close)
	conn, err = client.Request(ctx, request.Clone())
	assert.NotNil(t, t, conn)
	assert.NoError(t, err)

	// Call refresh from eventFactory. We are expecting updated value in the context
	eventFactoryCl.callRefresh()
	<-syncChan
}

type eventFactoryClient struct {
	ctx context.Context
	ch  chan<- struct{}
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
	go func() {
		s.ch <- struct{}{}
		eventFactory.Close()
	}()
}

func (s *eventFactoryClient) callRefresh() {
	eventFactory := begin.FromContext(s.ctx)
	go func() {
		s.ch <- struct{}{}
		eventFactory.Request()
	}()
}

type contextKey struct{}

type checkContextClient struct {
	t             *testing.T
	expectedValue string
}

func (c *checkContextClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	assert.Equal(c.t, c.expectedValue, ctx.Value(contextKey{}))
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *checkContextClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func (c *checkContextClient) setExpectedValue(value string) {
	c.expectedValue = value
}
