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

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/expire"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/serializectx"
)

type expireNSEServer struct {
	expireManager *expire.Manager
	nseExpiration time.Duration
}

// NewNetworkServiceEndpointRegistryServer creates a new NetworkServiceServer chain element that implements unregister
// of expired connections for the subsequent chain elements.
func NewNetworkServiceEndpointRegistryServer(ctx context.Context, nseExpiration time.Duration) registry.NetworkServiceEndpointRegistryServer {
	return &expireNSEServer{
		expireManager: expire.NewManager(ctx),
		nseExpiration: nseExpiration,
	}
}

func (s *expireNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	clockTime := clock.FromContext(ctx)
	logger := log.FromContext(ctx).WithField("expireNSEServer", "Register")

	s.expireManager.Stop(nse.Name)

	expirationTime := clockTime.Now().Add(s.nseExpiration)
	if nse.ExpirationTime != nil {
		if nseExpirationTime := nse.ExpirationTime.AsTime().Local(); nseExpirationTime.Before(expirationTime) {
			expirationTime = nseExpirationTime
		}
	}
	nse.ExpirationTime = timestamppb.New(expirationTime)

	reg, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		s.expireManager.Start(nse.Name)
		return nil, err
	}

	unregisterNSE := reg.Clone()
	if unregisterNSE.Name != nse.Name {
		s.expireManager.Delete(nse.Name)
	}

	s.expireManager.New(
		serializectx.GetExecutor(ctx, unregisterNSE.Name),
		unregisterNSE.Name,
		unregisterNSE.ExpirationTime.AsTime().Local(),
		func(unregisterCtx context.Context) {
			if _, unregisterErr := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(unregisterCtx, unregisterNSE); unregisterErr != nil {
				logger.Errorf("failed to unregister expired endpoint: %s %s", unregisterNSE.Name, unregisterErr.Error())
			}
		},
	)

	return reg, nil
}

func (s *expireNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *expireNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("expireNSEServer", "Unregister")

	if s.expireManager.Stop(nse.Name) {
		if _, err := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse); err != nil {
			s.expireManager.Start(nse.Name)
			return nil, err
		}
		s.expireManager.Delete(nse.Name)
	} else {
		logger.Warnf("endpoint has been already unregistered: %s", nse.Name)
	}

	return new(empty.Empty), nil
}
