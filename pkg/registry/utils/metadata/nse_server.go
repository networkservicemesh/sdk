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

package metadata

import (
	"context"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type metadataNSEServer struct {
	Map genericsync.Map[string, *metaData]
}

// NewNetworkServiceEndpointServer - enables per nse.Name metadata for the nse registry client.
func NewNetworkServiceEndpointServer() registry.NetworkServiceEndpointRegistryServer {
	return &metadataNSEServer{}
}

func (m *metadataNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	nseName := nse.GetName()
	_, isEstablished := m.Map.Load(nseName)

	n, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(store(ctx, nseName, &m.Map), nse)
	if err != nil {
		if !isEstablished {
			del(ctx, nseName, &m.Map)
			log.FromContext(ctx).
				WithField("metadata", "nse server").
				WithField("nseName", nseName).
				Debugf("metadata deleted")
		}
		return nil, err
	}
	return n, nil
}

func (m *metadataNSEServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	nseName := query.GetNetworkServiceEndpoint().GetName()
	s = streamcontext.NetworkServiceEndpointRegistryFindServer(load(s.Context(), nseName, &m.Map), s)
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, s)
}

func (m *metadataNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	delCtx := del(ctx, nse.GetName(), &m.Map)
	log.FromContext(ctx).
		WithField("metadata", "nse server").
		WithField("nse name", nse.GetName()).
		Debugf("metadata deleted")
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(delCtx, nse)
}
