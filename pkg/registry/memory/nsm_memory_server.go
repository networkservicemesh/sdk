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

package memory

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type nsmMemoryNetworkServerRegistry struct {
	storage *Storage
	nsmName string
}

func (n *nsmMemoryNetworkServerRegistry) RegisterNSM(ctx context.Context, nsm *registry.NetworkServiceManager) (*registry.NetworkServiceManager, error) {
	nsm.Name = n.nsmName
	n.storage.NetworkServiceManagers.Store(nsm.Name, nsm)
	return next.NSMRegistryServer(ctx).RegisterNSM(ctx, nsm)
}

func (n *nsmMemoryNetworkServerRegistry) GetEndpoints(context.Context, *empty.Empty) (*registry.NetworkServiceEndpointList, error) {
	result := new(registry.NetworkServiceEndpointList)
	n.storage.NetworkServiceEndpoints.Range(func(_ string, v *registry.NetworkServiceEndpoint) bool {
		if v.NetworkServiceManagerName == n.nsmName {
			result.NetworkServiceEndpoints = append(result.NetworkServiceEndpoints, v)
		}
		return true
	})
	return result, nil
}

// NewNSMRegistryServer returns new instance of NsmRegistryServer based on resource client
func NewNSMRegistryServer(nsmName string, storage *Storage) registry.NsmRegistryServer {
	return &nsmMemoryNetworkServerRegistry{
		storage: storage,
		nsmName: nsmName,
	}
}

var _ registry.NsmRegistryServer = &nsmMemoryNetworkServerRegistry{}
