// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type nseServer struct {
	timers timerMap
}

func (n *nseServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	r, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}
	if nse.ExpirationTime == nil {
		nse.ExpirationTime = &timestamp.Timestamp{}
	}
	if nse.ExpirationTime.Seconds != 0 || nse.ExpirationTime.Nanos != 0 {
		duration := time.Until(time.Unix(nse.ExpirationTime.Seconds, int64(nse.ExpirationTime.Nanos)))
		timer := time.AfterFunc(duration, func() {
			_, _ = next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
		})
		if t, load := n.timers.LoadOrStore(nse.Name, timer); load {
			timer.Stop()
			t.Reset(duration)
		}
	}
	return r, nil
}

func (n *nseServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, s)
}

func (n *nseServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
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
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return &nseServer{}
}
