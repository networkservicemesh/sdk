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

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
)

type memoryServiceDiscoveryServer struct {
	memory Memory
}

func (d *memoryServiceDiscoveryServer) FindNetworkService(ctx context.Context, req *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	service := d.memory.NetworkServices().Get(req.NetworkServiceName)
	if service == nil {
		return nil, errors.Errorf("network service %v is not found", req.NetworkServiceName)
	}
	NSMs := map[string]*registry.NetworkServiceManager{}
	NSEs := d.memory.NetworkServiceEndpoints().GetAllByFilter(func(endpoint *registry.NetworkServiceEndpoint) bool {
		return endpoint.NetworkServiceName == req.NetworkServiceName
	})
	for _, nse := range NSEs {
		nsm := d.memory.NetworkServiceManagers().Get(nse.NetworkServiceManagerName)
		NSMs[nsm.Name] = nsm
	}
	return &registry.FindNetworkServiceResponse{
		Payload: service.GetPayload(),
		NetworkService: &registry.NetworkService{
			Name:    service.GetName(),
			Payload: service.GetPayload(),
			Matches: service.Matches,
		},
		NetworkServiceManagers:  NSMs,
		NetworkServiceEndpoints: NSEs,
	}, nil
}

func NewNetworkServiceDiscoveryServer(memory Memory) registry.NetworkServiceDiscoveryServer {
	return &memoryServiceDiscoveryServer{
		memory: memory,
	}
}

var _ registry.NetworkServiceDiscoveryServer = &memoryServiceDiscoveryServer{}
