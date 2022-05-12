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
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dial"
	"github.com/networkservicemesh/sdk/pkg/registry/common/heal"
	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
	"github.com/networkservicemesh/sdk/pkg/registry/common/retry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
)

// NewNetworkServiceRegistryClient creates a new NewNetworkServiceRegistryClient that can be used for NS registration.
func NewNetworkServiceRegistryClient(ctx context.Context, opts ...Option) registry.NetworkServiceRegistryClient {
	clientOpts := &clientOptions{
		nsClientURLResolver: null.NewNetworkServiceRegistryClient(),
	}
	for _, opt := range opts {
		opt(clientOpts)
	}

	return chain.NewNetworkServiceRegistryClient(
		append(
			[]registry.NetworkServiceRegistryClient{
				begin.NewNetworkServiceRegistryClient(),
				retry.NewNetworkServiceRegistryClient(ctx),
				heal.NewNetworkServiceRegistryClient(ctx),
				clientOpts.nsClientURLResolver,
				clientconn.NewNetworkServiceRegistryClient(),
				dial.NewNetworkServiceRegistryClient(ctx,
					dial.WithDialOptions(clientOpts.dialOptions...),
					dial.WithDialTimeout(time.Second),
				),
			},
			append(
				clientOpts.nsAdditionalFunctionality,
				connect.NewNetworkServiceRegistryClient(),
			)...,
		)...,
	)
}
