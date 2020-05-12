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

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type nsmClientToServer struct {
	client registry.NsmRegistryClient
}

func (n *nsmClientToServer) RegisterNSM(ctx context.Context, r *registry.NetworkServiceManager) (*registry.NetworkServiceManager, error) {
	return n.client.RegisterNSM(ctx, r)
}

func (n *nsmClientToServer) GetEndpoints(ctx context.Context, e *empty.Empty) (*registry.NetworkServiceEndpointList, error) {
	return n.client.GetEndpoints(ctx, e)
}

// NewNSMClientToServer - returns a registry.NsmRegistryServer wrapped around the supplied client
func NewNSMClientToServer(client registry.NsmRegistryClient) registry.NsmRegistryServer {
	return &nsmClientToServer{client: client}
}
