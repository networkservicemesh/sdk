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

package defaultexpire

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type defaultexpireNSEServer struct {
	ctx                  context.Context
	defaultNSEExpiration time.Duration
}

// NewNetworkServiceEndpointRegistryServer creates a new NetworkServiceServer chain element that sets the default
// expiration time.
func NewNetworkServiceEndpointRegistryServer(ctx context.Context, defaultNSEExpiration time.Duration) registry.NetworkServiceEndpointRegistryServer {
	return &defaultexpireNSEServer{
		ctx:                  ctx,
		defaultNSEExpiration: defaultNSEExpiration,
	}
}

func (s *defaultexpireNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if nse.GetExpirationTime() == nil {
		nse.ExpirationTime = timestamppb.New(clock.FromContext(ctx).Now().Add(s.defaultNSEExpiration).Local())
		log.FromContext(ctx).Infof("default expiration time %v was set for %v", s.defaultNSEExpiration, nse.GetName())
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *defaultexpireNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *defaultexpireNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
