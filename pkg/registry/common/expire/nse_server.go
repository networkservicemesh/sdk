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
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/expire"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/serializectx"
)

type expireNSEServer struct {
	ctx           context.Context
	nseExpiration time.Duration
	timers        expire.TimerMap
}

// NewNetworkServiceEndpointRegistryServer creates a new NetworkServiceServer chain element that implements unregister
// of expired connections for the subsequent chain elements.
// For the algorithm please see /pkg/tools/expire/README.md.
func NewNetworkServiceEndpointRegistryServer(ctx context.Context, nseExpiration time.Duration) registry.NetworkServiceEndpointRegistryServer {
	return &expireNSEServer{
		ctx:           ctx,
		nseExpiration: nseExpiration,
	}
}

func (s *expireNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	clockTime := clock.FromContext(ctx)
	logger := log.FromContext(ctx).WithField("expireNSEServer", "Register")

	// 1. Try to load and stop timer.
	t, loaded := s.timers.Load(nse.Name)
	stopped := loaded && t.Stop()

	expirationTime := clockTime.Now().Add(s.nseExpiration)
	if nse.ExpirationTime != nil {
		if nseExpirationTime := nse.ExpirationTime.AsTime().Local(); nseExpirationTime.Before(expirationTime) {
			expirationTime = nseExpirationTime
		}
	}
	nse.ExpirationTime = timestamppb.New(expirationTime)

	// 2. Send Register event.
	reg, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		if stopped {
			// 2.1. Reset timer if Register event has failed and timer has been successfully stopped.
			t.Reset(clockTime.Until(t.ExpirationTime))
		}
		return nil, err
	}

	// 3. Delete the old timer.
	s.timers.Delete(nse.Name)

	// 4. Create a new timer.
	t, err = s.newTimer(ctx, reg.Clone())
	if err != nil {
		// 4.1. If we have failed to create a new timer, Unregister the endpoint.
		if _, unregisterErr := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, reg); unregisterErr != nil {
			logger.Errorf("failed to unregister endpoint on error: %s %s", reg.Name, unregisterErr.Error())
		}
		return nil, err
	}

	// 5. Store timer.
	s.timers.Store(reg.Name, t)

	return reg, nil
}

func (s *expireNSEServer) newTimer(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*expire.Timer, error) {
	clockTime := clock.FromContext(ctx)
	logger := log.FromContext(ctx).WithField("expireNSEServer", "newTimer")

	// 1. Try to get executor from the context.
	executor := serializectx.GetExecutor(ctx, nse.Name)
	if executor == nil {
		// 1.1. Fail if there is no executor.
		return nil, errors.Errorf("failed to get executor from context")
	}

	// 2. Compute expiration time.
	expirationTime := nse.ExpirationTime.AsTime().Local()

	// 3. Create timer.
	var t *expire.Timer
	t = &expire.Timer{
		ExpirationTime: expirationTime,
		Timer: clockTime.AfterFunc(clockTime.Until(expirationTime), func() {
			// 3.1. All the timer action should be executed under the `executor.AsyncExec`.
			executor.AsyncExec(func() {
				// 3.2. Timer has probably been stopped and deleted or replaced with a new one, so we need to check it
				// before deleting.
				if tt, ok := s.timers.Load(nse.Name); !ok || tt != t {
					// 3.2.1. This timer has been stopped, nothing to do.
					return
				}

				// 3.3. Delete timer.
				s.timers.Delete(nse.Name)

				// 3.4. Since `s.ctx` lives with the application, we need to create a new context with event scope
				// lifetime to prevent leaks.
				unregisterCtx, cancel := context.WithCancel(s.ctx)
				defer cancel()

				// 3.5. Unregister expired endpoint.
				if _, err := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(unregisterCtx, nse); err != nil {
					logger.Errorf("failed to unregister expired endpoint: %s %s", nse.Name, err.Error())
				}
			})
		}),
	}

	return t, nil
}

func (s *expireNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *expireNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("expireNSEServer", "Unregister")

	// 1. Check if we have a timer.
	t, ok := s.timers.LoadAndDelete(nse.Name)
	if !ok {
		// 1.1. If there is no timer, there is nothing to do.
		logger.Warnf("endpoint has been already unregistered: %s", nse.Name)
		return new(empty.Empty), nil
	}

	// 2. Stop it.
	t.Stop()

	// 3. Unregister endpoint.
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
