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

package translation

import "github.com/networkservicemesh/api/pkg/api/networkservice"

// NewNSMgrClient returns a new translation client for the NSMgr
func NewNSMgrClient() networkservice.NetworkServiceClient {
	return new(Builder).
		WithRequestOptions(
			ReplacePath(),
		).
		WithConnectionOptions(
			WithMechanism(),
			WithContext(),
			WithPath(),
		).
		Build()
}

// NewInterposeClient returns a new translation client for interpose NSE
func NewInterposeClient() networkservice.NetworkServiceClient {
	return new(Builder).
		WithRequestOptions(
			ReplaceMechanism(),
			ReplacePath(),
			ReplaceMechanismPreferences(),
		).
		WithConnectionOptions(
			WithContext(),
			WithPath(),
		).
		Build()
}

// NewPassThroughClient returns a new translation client for pass through NSE
func NewPassThroughClient(networkService, networkServiceEndpointName string) networkservice.NetworkServiceClient {
	return new(Builder).
		WithRequestOptions(
			func(request *networkservice.NetworkServiceRequest, _ *networkservice.Connection) {
				request.Connection.NetworkService = networkService
				request.Connection.NetworkServiceEndpointName = networkServiceEndpointName
			},
			ReplaceMechanismPreferences(),
		).
		WithConnectionOptions(
			WithContext(),
			WithPath(),
		).
		Build()
}
