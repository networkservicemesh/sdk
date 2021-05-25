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
	"context"
	"net/url"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/chains/connectto"
	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
	"github.com/networkservicemesh/sdk/pkg/registry/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
)

// NSOption is an option pattern for NewNetworkServiceRegistryClient
type NSOption func(nsOpts *nsClientOptions)

// WithNSAdditionalFunctionality sets additional functionality
func WithNSAdditionalFunctionality(additionalFunctionality ...registry.NetworkServiceRegistryClient) NSOption {
	return func(nsOpts *nsClientOptions) {
		nsOpts.additionalFunctionality = additionalFunctionality
	}
}

// WithNSDialOptions sets dial options
func WithNSDialOptions(dialOptions ...grpc.DialOption) NSOption {
	return func(nsOpts *nsClientOptions) {
		nsOpts.dialOptions = dialOptions
	}
}

type nsClientOptions struct {
	additionalFunctionality []registry.NetworkServiceRegistryClient
	dialOptions             []grpc.DialOption
}

// NewNetworkServiceRegistryClient creates a new NewNetworkServiceRegistryClient that can be used for NS registration.
func NewNetworkServiceRegistryClient(ctx context.Context, connectTo *url.URL, opts ...NSOption) registry.NetworkServiceRegistryClient {
	nsOpts := new(nsClientOptions)
	for _, opt := range opts {
		opt(nsOpts)
	}

	var additionalFunctionality registry.NetworkServiceRegistryClient
	if len(nsOpts.additionalFunctionality) > 0 {
		additionalFunctionality = chain.NewNetworkServiceRegistryClient(additionalFunctionality)
	} else {
		additionalFunctionality = null.NewNetworkServiceRegistryClient()
	}

	return chain.NewNetworkServiceRegistryClient(
		serialize.NewNetworkServiceRegistryClient(),
		additionalFunctionality,
		adapters.NetworkServiceServerToClient(
			connectto.NewNetworkServiceRegistryServer(ctx, connectTo, nsOpts.dialOptions...),
		),
	)
}
