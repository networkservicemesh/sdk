// Copyright (c) 2020 Cisco Systems, Inc.
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
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/sdk/pkg/registry/core/debug"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/trace"
)

// NewNetworkServiceRegistryServer - creates a chain of servers
func NewNetworkServiceRegistryServer(servers ...registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	server := trace.NewNetworkServiceRegistryServer
	if logrus.GetLevel() == logrus.DebugLevel {
		server = debug.NewNetworkServiceRegistryServer
	}

	return next.NewNetworkServiceRegistryServer(next.NewWrappedNetworkServiceRegistryServer(server, servers...))
}

// NewNetworkServiceRegistryClient - creates a chain of clients
func NewNetworkServiceRegistryClient(clients ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	client := trace.NewNetworkServiceRegistryClient
	if logrus.GetLevel() == logrus.DebugLevel {
		client = debug.NewNetworkServiceRegistryClient
	}

	return next.NewNetworkServiceRegistryClient(next.NewWrappedNetworkServiceRegistryClient(client, clients...))
}
