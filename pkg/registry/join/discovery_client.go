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

package join

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
)

type discoveryRegistryClient struct {
	clients []registry.NetworkServiceDiscoveryClient
}

func (d *discoveryRegistryClient) FindNetworkService(ctx context.Context, in *registry.FindNetworkServiceRequest, opts ...grpc.CallOption) (*registry.FindNetworkServiceResponse, error) {
	var result = &registry.FindNetworkServiceResponse{
		NetworkServiceManagers: map[string]*registry.NetworkServiceManager{},
	}
	var set = map[string]*registry.NetworkServiceEndpoint{}
	for _, c := range d.clients {
		resp, err := c.FindNetworkService(ctx, in, opts...)
		if err != nil {
			return nil, err
		}
		if result.NetworkService == nil {
			result.NetworkService = resp.NetworkService
		}
		if result.Payload == "" {
			result.Payload = resp.Payload
		}
		for _, nse := range resp.NetworkServiceEndpoints {
			if _, ok := set[nse.Name]; ok {
				continue
			}
			set[nse.Name] = nse
			result.NetworkServiceEndpoints = append(result.NetworkServiceEndpoints, nse)
		}
		for _, nsm := range resp.NetworkServiceManagers {
			result.NetworkServiceManagers[nsm.Name] = nsm
		}
	}
	return result, nil
}

func NewDiscoveryClient(clients ...registry.NetworkServiceDiscoveryClient) registry.NetworkServiceDiscoveryClient {
	return &discoveryRegistryClient{clients: clients}
}

var _ registry.NetworkServiceDiscoveryClient = &discoveryRegistryClient{}
