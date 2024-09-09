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

type metaDataNSClient struct {
	Map genericsync.Map[string, *metaData]
}

// NewNetworkServiceClient - enables per ns.Name metadata for the ns registry client.
func NewNetworkServiceClient() registry.NetworkServiceRegistryClient {
	return &metaDataNSClient{}
}

func (m *metaDataNSClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	nsName := ns.GetName()
	_, isEstablished := m.Map.Load(nsName)

	n, err := next.NetworkServiceRegistryClient(ctx).Register(store(ctx, nsName, &m.Map), ns, opts...)
	if err != nil {
		if !isEstablished {
			del(ctx, nsName, &m.Map)
			log.FromContext(ctx).
				WithField("metadata", "ns client").
				WithField("nsName", nsName).
				Debugf("metadata deleted")
		}
		return nil, err
	}
	return n, nil
}

func (m *metaDataNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	nsName := query.GetNetworkService().GetName()
	return next.NetworkServiceRegistryClient(ctx).Find(load(ctx, nsName, &m.Map), query, opts...)
}

func (m *metaDataNSClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	delCtx := del(ctx, ns.GetName(), &m.Map)
	log.FromContext(ctx).
		WithField("metadata", "ns client").
		WithField("ns name", ns.GetName()).
		Debugf("metadata deleted")

	return next.NetworkServiceRegistryClient(ctx).Unregister(delCtx, ns, opts...)
}
