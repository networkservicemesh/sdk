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

// Package streamutils provides utils for registry streams
package streamutils

import "github.com/networkservicemesh/api/pkg/api/registry"

// ReadNetworkServiceList read list of NetworkServices from passed stream
func ReadNetworkServiceList(stream registry.NetworkServiceRegistry_FindClient) []*registry.NetworkService {
	var result []*registry.NetworkService
	for msg, err := stream.Recv(); true; msg, err = stream.Recv() {
		if err != nil {
			break
		}
		result = append(result, msg)
	}
	return result
}

// ReadNetworkServiceEndpointList read list of NetworkServiceEndpoints from passed stream
func ReadNetworkServiceEndpointList(stream registry.NetworkServiceEndpointRegistry_FindClient) []*registry.NetworkServiceEndpoint {
	var result []*registry.NetworkServiceEndpoint
	for msg, err := stream.Recv(); true; msg, err = stream.Recv() {
		if err != nil {
			break
		}
		result = append(result, msg)
	}
	return result
}
