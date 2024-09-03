// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

package setregistrationtime

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

type setregtimeNSEServer struct{}

// NewNetworkServiceEndpointRegistryServer creates a new NetworkServiceServer chain element that sets initial
// registration time.
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return &setregtimeNSEServer{}
}

func (r *setregtimeNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if v, ok := load(ctx); ok {
		nse.InitialRegistrationTime = v
	} else {
		if nse.GetInitialRegistrationTime() == nil {
			nse.InitialRegistrationTime = timestamppb.New(clock.FromContext(ctx).Now())
		}
		store(ctx, nse.GetInitialRegistrationTime())
	}

	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (r *setregtimeNSEServer) Find(q *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(q, s)
}

func (r *setregtimeNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	deleteTime(ctx)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
