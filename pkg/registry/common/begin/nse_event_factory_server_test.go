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

	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

// This test reproduces the situation when Unregister and Register were called at the same time
func TestRefreshDuringUnregister_Server(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	syncChan := make(chan struct{})
	checkCtxServ := &checkContextServer{t: t}
	eventFactoryServ := &eventFactoryServer{ch: syncChan}
	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		checkCtxServ,
		eventFactoryServ,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set any value to context
	ctx = context.WithValue(ctx, contextKey{}, "value_1")
	checkCtxServ.setExpectedValue("value_1")

	// Do Register with this context
	nse := &registry.NetworkServiceEndpoint{
		Name: "1",
	}
	conn, err := server.Register(ctx, nse.Clone())
	assert.NotNil(t, t, conn)
	assert.NoError(t, err)

	// Change context value before refresh
	ctx = context.WithValue(ctx, contextKey{}, "value_2")
	checkCtxServ.setExpectedValue("value_2")

	// Call Unregister from eventFactory
	eventFactoryServ.callUnregister()
	<-syncChan

	// Call refresh (should be called at the same time as Unregister)
	conn, err = server.Register(ctx, nse.Clone())
	assert.NotNil(t, t, conn)
	assert.NoError(t, err)

	// Call refresh from eventFactory. We are expecting updated value in the context
	eventFactoryServ.callRefresh()
	<-syncChan
}

type eventFactoryServer struct {
	registry.NetworkServiceEndpointRegistryServer
	ctx context.Context
	ch  chan<- struct{}
}

func (e *eventFactoryServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	e.ctx = ctx
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (e *eventFactoryServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*emptypb.Empty, error) {
	// Wait to be sure that reregister was called
	time.Sleep(time.Millisecond * 100)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}

func (e *eventFactoryServer) callUnregister() {
	eventFactory := begin.FromContext(e.ctx)
	go func() {
		e.ch <- struct{}{}
		eventFactory.Unregister()
	}()
}

func (e *eventFactoryServer) callRefresh() {
	eventFactory := begin.FromContext(e.ctx)
	go func() {
		e.ch <- struct{}{}
		eventFactory.Register()
	}()
}

type checkContextServer struct {
	registry.NetworkServiceEndpointRegistryServer
	t             *testing.T
	expectedValue string
}

func (c *checkContextServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	assert.Equal(c.t, c.expectedValue, ctx.Value(contextKey{}))
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (c *checkContextServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*emptypb.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}

func (c *checkContextServer) setExpectedValue(value string) {
	c.expectedValue = value
}
