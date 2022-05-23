// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022 Cisco and/or its affiliates.
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

// Package clientinfo provides a chain element that adds pod, node and cluster names to request
package clientinfo

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clientinfo"
)

type clientInfoServer struct{}

// NewNetworkServiceEndpointRegistryServer - creates a new registry.NetworkServiceEndpointRegistryServer chain element
// that adds pod, node and cluster names to endpoint labels from corresponding environment variables
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return &clientInfoServer{}
}

func (c *clientInfoServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	for _, v := range nse.NetworkServiceLabels {
		if v.Labels == nil {
			v.Labels = make(map[string]string)
		}
		clientinfo.AddClientInfo(ctx, v.Labels)
	}

	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (c *clientInfoServer) Find(q *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(q, s)
}

func (c *clientInfoServer) Unregister(ctx context.Context, ns *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, ns)
}
