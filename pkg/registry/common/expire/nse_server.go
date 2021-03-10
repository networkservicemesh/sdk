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

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type expireNSEServer struct {
	ctx           context.Context
	nseExpiration time.Duration
	timers        unregisterTimerMap
}

type unregisterTimer struct {
	expirationTime time.Time
	timer          *time.Timer
	ch             <-chan struct{}
}

// NewNetworkServiceEndpointRegistryServer creates a new NetworkServiceServer chain element that implements unregister
// of expired connections for the subsequent chain elements.
// WARNING: `expire` uses ctx as a context for the Unregister, so if there are any chain elements setting some data in
// context in chain before the `expire`, these changes won't appear in the Unregister context.
func NewNetworkServiceEndpointRegistryServer(ctx context.Context, nseExpiration time.Duration) registry.NetworkServiceEndpointRegistryServer {
	return &expireNSEServer{
		ctx:           ctx,
		nseExpiration: nseExpiration,
	}
}

func (n *expireNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	t, loaded := n.timers.Load(nse.Name)
	// TODO: this is totally incorrect if there are concurrent events for the same nse.Name, think about adding
	//       serialize into the registry chain
	stopped := loaded && t.timer.Stop()

	if loaded && !stopped {
		<-t.ch
	}

	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		if stopped {
			t.timer.Reset(time.Until(t.expirationTime))
		}
		return nil, err
	}

	expirationTime := time.Now().Add(n.nseExpiration)
	if resp.ExpirationTime != nil {
		if respExpirationTime := resp.ExpirationTime.AsTime().Local(); respExpirationTime.Before(expirationTime) {
			expirationTime = respExpirationTime
		}
	}
	resp.ExpirationTime = timestamppb.New(expirationTime)

	n.timers.Store(resp.Name, n.newTimer(ctx, expirationTime, resp.Clone()))

	return resp, nil
}

func (n *expireNSEServer) newTimer(ctx context.Context, expirationTime time.Time, nse *registry.NetworkServiceEndpoint) *unregisterTimer {
	logger := log.FromContext(ctx).WithField("expireNSEServer", "newTimer")

	ch := make(chan struct{})
	return &unregisterTimer{
		expirationTime: expirationTime,
		timer: time.AfterFunc(time.Until(expirationTime), func() {
			unregisterCtx, cancel := context.WithCancel(n.ctx)
			defer cancel()

			if _, err := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(unregisterCtx, nse); err != nil {
				logger.Errorf("failed to unregister expired endpoint: %s %s", nse.Name, err.Error())
			}

			n.timers.Delete(nse.Name)
			close(ch)
		}),
		ch: ch,
	}
}

func (n *expireNSEServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, s)
}

func (n *expireNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("expireNSEServer", "Unregister")

	t, ok := n.timers.LoadAndDelete(nse.Name)
	// TODO: this is totally incorrect if there are concurrent events for the same nse.Name, think about adding
	//       serialize into the registry chain
	if ok && !t.timer.Stop() {
		<-t.ch
		ok = false
	}
	if !ok {
		logger.Warnf("endpoint has been already unregistered: %s", nse.Name)
		return new(empty.Empty), nil
	}

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
