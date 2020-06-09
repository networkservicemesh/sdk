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

package nsmreg

import (
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
)

func newNSMregistryServer(servers ...registry.NsmRegistryServer) registry.NsmRegistryServer {
	var result []registry.NsmRegistryServer
	for _, r := range servers {
		if r != nil {
			result = append(result, r)
		}
	}
	return chain.NewNSMRegistryServer(result...)
}
func newRegistryServer(servers ...registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	var result []registry.NetworkServiceRegistryServer
	for _, r := range servers {
		if r != nil {
			result = append(result, r)
		}
	}
	return chain.NewNetworkServiceRegistryServer(result...)
}
func newDiscoveryServer(servers ...registry.NetworkServiceDiscoveryServer) registry.NetworkServiceDiscoveryServer {
	var result []registry.NetworkServiceDiscoveryServer
	for _, r := range servers {
		if r != nil {
			result = append(result, r)
		}
	}
	return chain.NewDiscoveryServer(result...)
}
