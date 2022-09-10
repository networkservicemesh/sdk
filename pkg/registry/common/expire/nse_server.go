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

package expire

import (
	"context"
	"time"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type expireNSEServer struct {
	defaultExpiration time.Duration
	ctx               context.Context
	genericsync.Map[string, context.CancelFunc]
}

// NewNetworkServiceEndpointRegistryServer creates a new NetworkServiceServer chain element that implements unregister
// of expired connections for the subsequent chain elements.
func NewNetworkServiceEndpointRegistryServer(ctx context.Context, opts ...Option) registry.NetworkServiceEndpointRegistryServer {
	var serverOptions = &options{}

	for _, opt := range opts {
		opt(serverOptions)
	}

	return &expireNSEServer{
		ctx:               ctx,
		defaultExpiration: serverOptions.defaultExpiration,
	}
}

func (s *expireNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	factory := begin.FromContext(ctx)
	timeClock := clock.FromContext(ctx)

	logger := log.FromContext(ctx).WithField("expireNSEServer", "Register")

	deadline, ok := ctx.Deadline()
	requestTimeout := timeClock.Until(deadline)
	if !ok {
		requestTimeout = 0
	}

	expirationTime := nse.GetExpirationTime().AsTime()
	if nse.GetExpirationTime() == nil {
		expirationTime = timeClock.Now().Add(s.defaultExpiration).Local()
		nse.ExpirationTime = timestamppb.New(expirationTime)
		logger.Infof("selected expiration time %v for %v", expirationTime, nse.GetName())
	}

	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}

	if nseExpirationTime := resp.GetExpirationTime().AsTime().Local(); nseExpirationTime.Before(expirationTime) {
		expirationTime = nseExpirationTime
		logger.Infof("selected expiration time %v for %v", expirationTime, resp.GetName())
	}

	expireContext, cancel := context.WithCancel(s.ctx)
	if v, ok := s.Map.LoadAndDelete(nse.GetName()); ok {
		v()
	}
	s.Map.Store(nse.GetName(), cancel)

	expireCh := timeClock.After(timeClock.Until(expirationTime.Local()) - requestTimeout)

	go func() {
		select {
		case <-expireContext.Done():
			return
		case <-expireCh:
			factory.Unregister(begin.CancelContext(expireContext))
		}
	}()

	return resp, nil
}

func (s *expireNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *expireNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if oldCancel, loaded := s.LoadAndDelete(nse.Name); loaded {
		oldCancel()
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
