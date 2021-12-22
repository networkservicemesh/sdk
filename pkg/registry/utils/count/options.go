// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package count

type options struct {
	totalRegisterCalls, totalUnregisterCalls, totalFindCalls *int32
}

// Option is an option pattern for NewNetworkServiceRegistryClient, NewNetworkServiceEndpointRegistryClient
type Option func(*options)

// WithTotalRegisterCalls sets pointer to number of Register calls
func WithTotalRegisterCalls(registerCount *int32) func(*options) {
	return func(o *options) {
		o.totalRegisterCalls = registerCount
	}
}

// WithTotalUnregisterCalls sets pointer to number of Unregister calls
func WithTotalUnregisterCalls(unregisterCount *int32) func(*options) {
	return func(o *options) {
		o.totalUnregisterCalls = unregisterCount
	}
}

// WithTotalFindCalls sets pointer to number of Find calls
func WithTotalFindCalls(findCalls *int32) func(*options) {
	return func(o *options) {
		o.totalFindCalls = findCalls
	}
}
