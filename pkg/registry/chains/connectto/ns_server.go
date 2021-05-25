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

package connectto

import (
	"context"
	"net/url"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
)

// NewNetworkServiceRegistryServer creates a new NS server chain to connect to the `connectTo` remote NS server
func NewNetworkServiceRegistryServer(ctx context.Context, connectTo *url.URL, dialOptions ...grpc.DialOption) registry.NetworkServiceRegistryServer {
	return chain.NewNetworkServiceRegistryServer(
		clienturl.NewNetworkServiceRegistryServer(connectTo),
		connect.NewNetworkServiceRegistryServer(ctx, func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient {
			return registry.NewNetworkServiceRegistryClient(cc)
		}, dialOptions...),
	)
}
