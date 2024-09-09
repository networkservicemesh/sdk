// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023-2024 Cisco and/or its affiliates.
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
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type expireNSEServer struct {
	defaultExpiration time.Duration
	ctx               context.Context
	genericsync.Map[string, context.CancelFunc]
}

// NewNetworkServiceEndpointRegistryServer creates a new NetworkServiceServer chain element that implements unregister
// of expired connections for the subsequent chain elements.
func NewNetworkServiceEndpointRegistryServer(ctx context.Context, opts ...Option) registry.NetworkServiceEndpointRegistryServer {
	serverOptions := &options{}

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

	// Select the min(tokenExpirationTime, peerExpirationTime, defaultExpirationTime)
	expirationTime, expirationTimeSelected := s.selectMinExpirationTime(ctx)

	// Update nse ExpirationTime if expirationTime is before
	if expirationTimeSelected && (nse.GetExpirationTime() == nil || expirationTime.Before(nse.GetExpirationTime().AsTime().Local())) {
		nse.ExpirationTime = timestamppb.New(expirationTime)
		logger.Infof("selected expiration time %v for %v", expirationTime, nse.GetName())
	}

	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}

	if nseExpirationTime := resp.GetExpirationTime().AsTime().Local(); expirationTimeSelected && nseExpirationTime.Before(expirationTime) {
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
			factory.Unregister(begin.CancelContext(expireContext), begin.ExtendContext(ctx))
		}
	}()

	return resp, nil
}

func (s *expireNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *expireNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if oldCancel, loaded := s.LoadAndDelete(nse.GetName()); loaded {
		oldCancel()
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

func (s *expireNSEServer) selectMinExpirationTime(ctx context.Context) (time.Time, bool) {
	timeClock := clock.FromContext(ctx)

	var expirationTime *time.Time
	if tokenExpirationTime := updatepath.ExpirationTimeFromContext(ctx); tokenExpirationTime != nil {
		expirationTime = tokenExpirationTime
	} else {
		log.FromContext(ctx).Warn("error during getting token expiration time from the context")
	}

	if _, peerExpirationTime, peerTokenErr := token.FromContext(ctx); peerTokenErr == nil {
		if expirationTime == nil || peerExpirationTime.Before(*expirationTime) {
			expirationTime = &peerExpirationTime
		}
	} else {
		log.FromContext(ctx).Warnf("error during getting peer expiration time from the context: %v", peerTokenErr)
	}

	if s.defaultExpiration != 0 {
		defaultExpirationTime := timeClock.Now().Add(s.defaultExpiration).Local()
		if expirationTime == nil || defaultExpirationTime.Before(*expirationTime) {
			expirationTime = &defaultExpirationTime
		}
	}
	if expirationTime == nil {
		return time.Time{}, false
	}

	return *expirationTime, true
}
