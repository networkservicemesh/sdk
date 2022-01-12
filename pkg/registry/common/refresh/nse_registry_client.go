// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

package refresh

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
	"github.com/networkservicemesh/sdk/pkg/tools/serializectx"
)

type refreshNSEClient struct {
	ctx        context.Context
	nseCancels cancelsMap
}

// NewNetworkServiceEndpointRegistryClient creates new NetworkServiceEndpointRegistryClient that will refresh expiration
// time for registered NSEs
func NewNetworkServiceEndpointRegistryClient(ctx context.Context) registry.NetworkServiceEndpointRegistryClient {
	return &refreshNSEClient{
		ctx: ctx,
	}
}

func (c *refreshNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	clockTime := clock.FromContext(ctx)
	logger := log.FromContext(ctx).WithField("refreshNSEClient", "Register")

	var expirationDuration time.Duration
	if nse.ExpirationTime != nil {
		expirationDuration = clockTime.Until(nse.ExpirationTime.AsTime().Local())
	}

	cancel, ok := c.nseCancels.LoadAndDelete(nse.Name)
	if ok {
		cancel()
	}

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	reg, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
	if err != nil {
		return nil, err
	}

	if reg.ExpirationTime != nil {
		refreshNSE := nse.Clone()
		refreshNSE.ExpirationTime = reg.ExpirationTime
		refreshNSE.InitialRegistrationTime = reg.InitialRegistrationTime

		cancel, err = c.startRefresh(ctx, refreshNSE, expirationDuration)
		if err != nil {
			unregisterCtx, cancelUnregister := postponeCtxFunc()
			defer cancelUnregister()

			if _, unregisterErr := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(unregisterCtx, reg); unregisterErr != nil {
				logger.Errorf("failed to unregister endpoint on error: %s %s", reg.Name, unregisterErr.Error())
			}
			return nil, err
		}

		c.nseCancels.Store(refreshNSE.Name, cancel)
	}

	return reg, err
}

func (c *refreshNSEClient) startRefresh(ctx context.Context, nse *registry.NetworkServiceEndpoint, expirationDuration time.Duration) (context.CancelFunc, error) {
	clockTime := clock.FromContext(ctx)
	logger := log.FromContext(ctx).WithField("refreshNSEClient", "startRefresh")

	executor := serializectx.GetExecutor(ctx, nse.Name)
	if executor == nil {
		return nil, errors.Errorf("failed to get executor from context")
	}

	expirationTime := nse.ExpirationTime.AsTime().Local()
	refreshCh := clockTime.After(2 * clockTime.Until(expirationTime) / 3)

	refreshCtx, refreshCancel := context.WithCancel(c.ctx)
	go func() {
		defer refreshCancel()
		for {
			select {
			case <-refreshCtx.Done():
				return
			case <-refreshCh:
				<-executor.AsyncExec(func() {
					if refreshCtx.Err() != nil {
						return
					}

					var registerCtx context.Context
					var cancel context.CancelFunc
					if expirationDuration != 0 {
						nse.ExpirationTime = timestamppb.New(clockTime.Now().Add(expirationDuration))
						registerCtx, cancel = clockTime.WithTimeout(refreshCtx, expirationDuration)
					} else {
						nse.ExpirationTime = nil
						registerCtx, cancel = context.WithCancel(refreshCtx)
					}
					defer cancel()

					reg, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(registerCtx, nse.Clone())
					if err != nil {
						logger.Errorf("failed to refresh endpoint registration: %s %s", nse.Name, err.Error())
						return
					}

					if reg.ExpirationTime == nil {
						logger.Warnf("received nil expiration time: %s", nse.Name)
						return
					}

					expirationTime = reg.ExpirationTime.AsTime().Local()
					refreshCh = clockTime.After(2 * clockTime.Until(expirationTime) / 3)
				})
			}
		}
	}()

	return refreshCancel, nil
}

func (c *refreshNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *refreshNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	if cancel, ok := c.nseCancels.LoadAndDelete(nse.Name); ok {
		cancel()
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}
