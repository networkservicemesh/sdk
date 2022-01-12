// Copyright (c) 2022 Doc.ai and/or its affiliates.
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

package connect

import (
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

// Option is an option pattern for NewNetworkServiceRegistryClient, NewNetworkServiceEndpointRegistryClient
type Option func(connectOpts *connectOptions)

// WithNSAdditionalFunctionality sets additional functionality
func WithNSAdditionalFunctionality(additionalFunctionality ...registry.NetworkServiceRegistryClient) Option {
	return func(connectOpts *connectOptions) {
		connectOpts.nsAdditionalFunctionality = additionalFunctionality
	}
}

// WithNSEAdditionalFunctionality sets additional functionality
func WithNSEAdditionalFunctionality(additionalFunctionality ...registry.NetworkServiceEndpointRegistryClient) Option {
	return func(connectOpts *connectOptions) {
		connectOpts.nseAdditionalFunctionality = additionalFunctionality
	}
}

// WithDialOptions sets dial options
func WithDialOptions(dialOptions ...grpc.DialOption) Option {
	return func(connectOpts *connectOptions) {
		connectOpts.dialOptions = dialOptions
	}
}

type connectOptions struct {
	nsAdditionalFunctionality  []registry.NetworkServiceRegistryClient
	nseAdditionalFunctionality []registry.NetworkServiceEndpointRegistryClient
	dialOptions                []grpc.DialOption
}
