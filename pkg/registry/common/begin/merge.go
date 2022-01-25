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

package begin

import "github.com/networkservicemesh/api/pkg/api/registry"

func mergeNSE(left, right *registry.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint {
	if left == nil || right == nil {
		return left
	}

	var result = right.Clone()

	result.Name = left.Name

	result.ExpirationTime = nil

	return result
}

func mergeNS(left, right *registry.NetworkService) *registry.NetworkService {
	if left == nil || right == nil {
		return left
	}

	var result = right.Clone()

	result.Name = left.Name

	return result
}
