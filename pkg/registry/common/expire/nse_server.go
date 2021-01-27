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
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
)

type expireNSEServer struct {
	timers        timerMap
	nseExpiration time.Duration
}

func (n *expireNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}

	expirationTime := time.Now().Add(n.nseExpiration)
	if resp.ExpirationTime != nil {
		if respExpirationTime := resp.ExpirationTime.AsTime().Local(); respExpirationTime.Before(expirationTime) {
			expirationTime = respExpirationTime
		}
	}
	resp.ExpirationTime = timestamppb.New(expirationTime)

	unregisterNSE := resp.Clone()

	exipreDuration := time.Until(expirationTime)

	timer := time.AfterFunc(exipreDuration, func() {
		unregisterCtx, cancel := context.WithTimeout(extend.WithValuesFromContext(context.Background(), ctx), exipreDuration)
		defer cancel()
		_, _ = next.NetworkServiceEndpointRegistryServer(unregisterCtx).Unregister(unregisterCtx, unregisterNSE)
	})
	if t, load := n.timers.LoadOrStore(resp.Name, timer); load {
		timer.Stop()
		t.Stop()
		t.Reset(time.Until(expirationTime))
	}

	return resp, nil
}

func (n *expireNSEServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, s)
}

func (n *expireNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
	if err != nil {
		return nil, err
	}

	if timer, ok := n.timers.Load(nse.Name); ok {
		timer.Stop()
	}

	return resp, nil
}

// NewNetworkServiceEndpointRegistryServer wraps passed NetworkServiceEndpointRegistryServer and monitor Network service endpoints
func NewNetworkServiceEndpointRegistryServer(nseExpiration time.Duration) registry.NetworkServiceEndpointRegistryServer {
	return &expireNSEServer{
		nseExpiration: nseExpiration,
	}
}
