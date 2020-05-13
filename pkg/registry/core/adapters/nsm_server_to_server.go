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

package adapters

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type nsmServerToClient struct {
	server registry.NsmRegistryServer
}

func (n *nsmServerToClient) RegisterNSM(ctx context.Context, in *registry.NetworkServiceManager, opts ...grpc.CallOption) (*registry.NetworkServiceManager, error) {
	return n.server.RegisterNSM(ctx, in)
}

func (n *nsmServerToClient) GetEndpoints(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*registry.NetworkServiceEndpointList, error) {
	return n.server.GetEndpoints(ctx, in)
}

// NewNSMServerToClient - returns a registry.NsmRegistryClient wrapped around the supplied server
func NewNSMServerToClient(client registry.NsmRegistryServer) registry.NsmRegistryClient {
	return &nsmServerToClient{server: client}
}
