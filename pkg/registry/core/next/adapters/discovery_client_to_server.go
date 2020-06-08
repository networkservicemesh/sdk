// Copyright (c) 2020 Cisco Systems, Inc.
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

package adapters

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type discoveryClientToServer struct {
	client registry.NetworkServiceDiscoveryClient
	next   func(ctx context.Context) registry.NetworkServiceDiscoveryServer
}

// NewDiscoveryClientToServer - returns a registry.NetworkServiceDiscoveryClient wrapped around the supplied client
func NewDiscoveryClientToServer(client registry.NetworkServiceDiscoveryClient, next func(ctx context.Context) registry.NetworkServiceDiscoveryServer) registry.NetworkServiceDiscoveryServer {
	return &discoveryClientToServer{client: client, next: next}
}

func (c *discoveryClientToServer) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	result, err := c.client.FindNetworkService(ctx, request)
	if err != nil || c.next == nil {
		return result, err
	}
	var nextResult *registry.FindNetworkServiceResponse
	nextResult, err = c.next(ctx).FindNetworkService(ctx, request)
	if err != nil {
		return nil, err
	}
	result.NetworkServiceEndpoints = append(result.NetworkServiceEndpoints, nextResult.NetworkServiceEndpoints...)
	for k, v := range nextResult.NetworkServiceManagers {
		result.NetworkServiceManagers[k] = v
	}
	return result, nil
}

// Implementation check
var _ registry.NetworkServiceDiscoveryServer = &discoveryClientToServer{}
