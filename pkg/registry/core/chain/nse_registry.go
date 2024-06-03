// Copyright (c) 2020-2023 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2024 Cisco Systems, Inc.
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

// Package chain provides API to make chains of registry elements
package chain

import (
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/trace"
	"github.com/networkservicemesh/sdk/pkg/tools/opentelemetry"
)

// NewNetworkServiceEndpointRegistryServer - creates a chain of servers
func NewNetworkServiceEndpointRegistryServer(servers ...registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
	if opentelemetry.IsEnabled() {
		return next.NewWrappedNetworkServiceEndpointRegistryServer(trace.NewNetworkServiceEndpointRegistryServer, servers...)
	}
	return next.NewNetworkServiceEndpointRegistryServer(servers...)
}

// NewNetworkServiceEndpointRegistryClient - creates a chain of clients
func NewNetworkServiceEndpointRegistryClient(clients ...registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
	if opentelemetry.IsEnabled() {
		return next.NewWrappedNetworkServiceEndpointRegistryClient(trace.NewNetworkServiceEndpointRegistryClient, clients...)
	}
	return next.NewNetworkServiceEndpointRegistryClient(clients...)
}
