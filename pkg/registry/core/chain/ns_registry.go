// Copyright (c) 2020-2023 Cisco Systems, Inc.
//
// Copyright (c) 2020-2023 Doc.ai and/or its affiliates.
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

package chain

import (
	"os"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/trace"
)

const disableTracingEnv = "NSM_DISABLE_TRACING"

// NewNetworkServiceRegistryServer - creates a chain of servers
func NewNetworkServiceRegistryServer(servers ...registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	if os.Getenv(disableTracingEnv) != "" {
		return next.NewNetworkServiceRegistryServer(servers...)
	}
	return next.NewWrappedNetworkServiceRegistryServer(trace.NewNetworkServiceRegistryServer, servers...)
}

// NewNetworkServiceRegistryClient - creates a chain of clients
func NewNetworkServiceRegistryClient(clients ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	if os.Getenv(disableTracingEnv) != "" {
		return next.NewNetworkServiceRegistryClient(clients...)
	}
	return next.NewWrappedNetworkServiceRegistryClient(trace.NewNetworkServiceRegistryClient, clients...)
}
