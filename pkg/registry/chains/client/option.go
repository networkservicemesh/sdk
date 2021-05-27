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

package client

import (
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
)

// Option is an option pattern for NewNetworkServiceRegistryClient, NewNetworkServiceEndpointRegistryClient
type Option func(clientOpts *clientOptions)

// WithNSAdditionalFunctionality sets additional functionality
func WithNSAdditionalFunctionality(additionalFunctionality ...registry.NetworkServiceRegistryClient) Option {
	return func(clientOpts *clientOptions) {
		clientOpts.nsAdditionalFunctionality = additionalFunctionality
	}
}

// WithNSEAdditionalFunctionality sets additional functionality
func WithNSEAdditionalFunctionality(additionalFunctionality ...registry.NetworkServiceEndpointRegistryClient) Option {
	return func(clientOpts *clientOptions) {
		clientOpts.nseAdditionalFunctionality = additionalFunctionality
	}
}

// WithDialOptions sets dial options
func WithDialOptions(dialOptions ...grpc.DialOption) Option {
	return func(clientOpts *clientOptions) {
		clientOpts.dialOptions = dialOptions
	}
}

type clientOptions struct {
	nsAdditionalFunctionality  []registry.NetworkServiceRegistryClient
	nseAdditionalFunctionality []registry.NetworkServiceEndpointRegistryClient
	dialOptions                []grpc.DialOption
}
