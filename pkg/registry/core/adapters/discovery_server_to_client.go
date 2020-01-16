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

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type discoveryServerToClient struct {
	server registry.NetworkServiceDiscoveryServer
}

// NewDiscoveryServerToClient - returns a new registry.NetworkServiceDiscoveryServer that is a wrapper around server
func NewDiscoveryServerToClient(server registry.NetworkServiceDiscoveryServer) registry.NetworkServiceDiscoveryClient {
	return &discoveryServerToClient{server: server}
}

func (s *discoveryServerToClient) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest, opts ...grpc.CallOption) (*registry.FindNetworkServiceResponse, error) {
	return s.server.FindNetworkService(ctx, request)
}
