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

// Package clusterinfo provides a chain element that appends clusterinfo labels into the request.
package clusterinfo

import (
	"context"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"gopkg.in/yaml.v2"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type clusterinfoNSEServer struct {
	configPath string
}

func (n *clusterinfoNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	m := make(map[string]string)

	if b, err := os.ReadFile(n.configPath); err == nil {
		_ = yaml.Unmarshal(b, &m)
	}

	for k, v := range m {
		for _, labels := range nse.GetNetworkServiceLabels() {
			if labels.GetLabels() == nil {
				labels.Labels = make(map[string]string)
			}
			labels.GetLabels()[k] = v
		}
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (n *clusterinfoNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (n *clusterinfoNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

// NewNetworkServiceEndpointRegistryServer - returns a new clusterinfo server that adds clusterinfo labels into nse registration.
func NewNetworkServiceEndpointRegistryServer(opts ...Option) registry.NetworkServiceEndpointRegistryServer {
	r := &clusterinfoNSEServer{
		configPath: "/etc/clusterinfo/config.yaml",
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}
