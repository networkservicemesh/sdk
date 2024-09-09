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

type metadataNSServer struct {
	Map genericsync.Map[string, *metaData]
}

// NewNetworkServiceServer - enables per ns.Name metadata for the ns registry server.
func NewNetworkServiceServer() registry.NetworkServiceRegistryServer {
	return &metadataNSServer{}
}

func (m *metadataNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	nsName := ns.GetName()
	_, isEstablished := m.Map.Load(nsName)

	n, err := next.NetworkServiceRegistryServer(ctx).Register(store(ctx, nsName, &m.Map), ns)
	if err != nil {
		if !isEstablished {
			del(ctx, nsName, &m.Map)
			log.FromContext(ctx).
				WithField("metadata", "ns server").
				WithField("nsName", nsName).
				Debugf("metadata deleted")
		}
		return nil, err
	}
	return n, nil
}

func (m *metadataNSServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	nsName := query.GetNetworkService().GetName()
	s = streamcontext.NetworkServiceRegistryFindServer(load(s.Context(), nsName, &m.Map), s)
	return next.NetworkServiceRegistryServer(s.Context()).Find(query, s)
}

func (m *metadataNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	delCtx := del(ctx, ns.GetName(), &m.Map)
	log.FromContext(ctx).
		WithField("metadata", "ns server").
		WithField("ns name", ns.GetName()).
		Debugf("metadata deleted")
	return next.NetworkServiceRegistryServer(ctx).Unregister(delCtx, ns)
}
