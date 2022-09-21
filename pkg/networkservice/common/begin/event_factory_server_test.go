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
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// This test reproduces the situation when Close and Request were called at the same time
// nolint:dupl
func TestRefreshDuringClose_Server(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	syncChan := make(chan struct{})
	checkCtxServ := &checkContextServer{t: t}
	eventFactoryServ := &eventFactoryServer{ch: syncChan}
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
	assert.NotNil(t, t, conn)
	assert.NoError(t, err)

	// Change context value before refresh Request
	ctx = context.WithValue(ctx, contextKey{}, "value_2")
	checkCtxServ.setExpectedValue("value_2")
	request.Connection = conn.Clone()

	// Call Close from eventFactory
	eventFactoryServ.callClose()
	<-syncChan

	// Call refresh  (should be called at the same time as Close)
	conn, err = server.Request(ctx, request.Clone())
	assert.NotNil(t, t, conn)
	assert.NoError(t, err)

	// Call refresh from eventFactory. We are expecting updated value in the context
	eventFactoryServ.callRefresh()
	<-syncChan
}

type eventFactoryServer struct {
	ctx context.Context
	ch  chan<- struct{}
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
	go func() {
		e.ch <- struct{}{}
		eventFactory.Close()
	}()
}

func (e *eventFactoryServer) callRefresh() {
	eventFactory := begin.FromContext(e.ctx)
	go func() {
		e.ch <- struct{}{}
		eventFactory.Request()
	}()
}

type checkContextServer struct {
	t             *testing.T
	expectedValue string
}

func (c *checkContextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	assert.Equal(c.t, c.expectedValue, ctx.Value(contextKey{}))
	return next.Server(ctx).Request(ctx, request)
}

func (c *checkContextServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

func (c *checkContextServer) setExpectedValue(value string) {
	c.expectedValue = value
}
