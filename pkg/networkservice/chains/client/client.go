// Copyright (c) 2021 Cisco and/or its affiliates.
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

// Package client provides a simple wrapper for building a NetworkServiceMeshClient
package client

import (
	"context"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/dial"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/trimpath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

// NewClient - returns case NSM client.
//             - ctx    - context for the lifecycle of the *Client* itself.  Cancel when discarding the client.
func NewClient(ctx context.Context, clientOpts ...Option) networkservice.NetworkServiceClient {
	var opts = &clientOptions{
		name:            "client-" + uuid.New().String(),
		authorizeClient: null.NewClient(),
		healClient:      null.NewClient(),
		refreshClient:   refresh.NewClient(ctx),
	}
	for _, opt := range clientOpts {
		opt(opts)
	}

	return chain.NewNetworkServiceClient(
		append(
			[]networkservice.NetworkServiceClient{
				updatepath.NewClient(opts.name),
				begin.NewClient(),
				metadata.NewClient(),
				opts.refreshClient,
				clienturl.NewClient(opts.clientURL),
				clientconn.NewClient(opts.cc),
				opts.healClient,
				dial.NewClient(ctx,
					dial.WithDialOptions(opts.dialOptions...),
					dial.WithDialTimeout(opts.dialTimeout),
				),
			},
			append(
				opts.additionalFunctionality,
				opts.authorizeClient,
				trimpath.NewClient(),
				connect.NewClient(),
			)...,
		)...,
	)
}
