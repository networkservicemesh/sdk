// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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
	"net/url"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
)

// Option is an option pattern for NewNetworkServiceRegistryClient, NewNetworkServiceEndpointRegistryClient
type Option func(clientOpts *clientOptions)

// WithClientURL sets client URL
func WithClientURL(u *url.URL) Option {
	return func(clientOpts *clientOptions) {
		clientOpts.nsClientURLResolver = clienturl.NewNetworkServiceRegistryClient(u)
		clientOpts.nseClientURLResolver = clienturl.NewNetworkServiceEndpointRegistryClient(u)
	}
}

// WithNSClientURLResolver sets ns client URL resolver
func WithNSClientURLResolver(c registry.NetworkServiceRegistryClient) Option {
	return func(clientOpts *clientOptions) {
		clientOpts.nsClientURLResolver = c
	}
}

// WithNSEClientURLResolver sets nse client URL resolver
func WithNSEClientURLResolver(c registry.NetworkServiceEndpointRegistryClient) Option {
	return func(clientOpts *clientOptions) {
		clientOpts.nseClientURLResolver = c
	}
}

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
	nsClientURLResolver        registry.NetworkServiceRegistryClient
	nseClientURLResolver       registry.NetworkServiceEndpointRegistryClient
	nsAdditionalFunctionality  []registry.NetworkServiceRegistryClient
	nseAdditionalFunctionality []registry.NetworkServiceEndpointRegistryClient
	dialOptions                []grpc.DialOption
}
