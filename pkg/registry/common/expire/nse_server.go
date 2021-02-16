// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package expire

import (
	"context"
	"time"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type expireNSEServer struct {
	ctx           context.Context
	nseExpiration time.Duration
	timers        unregisterTimerMap
}

type unregisterTimer struct {
	expirationTime time.Time
	cancel         context.CancelFunc
	timer          *time.Timer
	executor       *serialize.Executor
}

// NewNetworkServiceEndpointRegistryServer wraps passed NetworkServiceEndpointRegistryServer and monitor Network service endpoints
func NewNetworkServiceEndpointRegistryServer(ctx context.Context, nseExpiration time.Duration) registry.NetworkServiceEndpointRegistryServer {
	return &expireNSEServer{
		ctx:           ctx,
		nseExpiration: nseExpiration,
	}
}

func (n *expireNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	t, loaded := n.timers.LoadAndDelete(nse.Name)
	stopped := loaded && t.timer.Stop()

	var expirationTime time.Time
	if stopped {
		expirationTime = t.expirationTime
	} else if loaded {
		<-t.executor.AsyncExec(t.cancel)
	}

	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		if stopped {
			t.timer.Reset(time.Until(expirationTime))
		}
		return nil, err
	}

	expirationTime = time.Now().Add(n.nseExpiration)
	if resp.ExpirationTime != nil {
		if respExpirationTime := resp.ExpirationTime.AsTime().Local(); respExpirationTime.Before(expirationTime) {
			expirationTime = respExpirationTime
		}
	}
	resp.ExpirationTime = timestamppb.New(expirationTime)

	t = n.newTimer(ctx, expirationTime, resp.Clone())
	n.timers.Store(resp.Name, t)

	return resp, nil
}

func (n *expireNSEServer) newTimer(
	ctx context.Context,
	expirationTime time.Time,
	nse *registry.NetworkServiceEndpoint,
) *unregisterTimer {
	unregisterCtx, cancel := context.WithCancel(n.ctx)
	executor := new(serialize.Executor)
	return &unregisterTimer{
		expirationTime: expirationTime,
		cancel:         cancel,
		executor:       executor,
		timer: time.AfterFunc(time.Until(expirationTime), func() {
			executor.AsyncExec(func() {
				if unregisterCtx.Err() != nil {
					return
				}
				defer cancel()

				_, _ = next.NetworkServiceEndpointRegistryServer(ctx).Unregister(unregisterCtx, nse)
			})
		}),
	}
}

func (n *expireNSEServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, s)
}

func (n *expireNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if t, ok := n.timers.LoadAndDelete(nse.Name); ok {
		if !t.timer.Stop() {
			<-t.executor.AsyncExec(t.cancel)
		}
	}

	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
