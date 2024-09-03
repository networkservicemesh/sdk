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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type metaDataNSEClient struct {
	Map genericsync.Map[string, *metaData]
}

// NewNetworkServiceEndpointClient - enables per nse.Name metadata for the nse registry client.
func NewNetworkServiceEndpointClient() registry.NetworkServiceEndpointRegistryClient {
	return &metaDataNSEClient{}
}

func (m *metaDataNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	nseName := nse.GetName()
	_, isEstablished := m.Map.Load(nseName)

	n, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(store(ctx, nseName, &m.Map), nse, opts...)
	if err != nil {
		if !isEstablished {
			del(ctx, nseName, &m.Map)
			log.FromContext(ctx).
				WithField("metadata", "nse client").
				WithField("nseName", nseName).
				Debugf("metadata deleted")
		}
		return nil, err
	}
	return n, nil
}

func (m *metaDataNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	nseName := query.GetNetworkServiceEndpoint().GetName()
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(load(ctx, nseName, &m.Map), query, opts...)
}

func (m *metaDataNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	delCtx := del(ctx, nse.GetName(), &m.Map)
	log.FromContext(ctx).
		WithField("metadata", "nse client").
		WithField("nse name", nse.GetName()).
		Debugf("metadata deleted")

	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(delCtx, nse, opts...)
}
