// Copyright (c) 2020 Cisco Systems, Inc.
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

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injectpeer"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// NewClient - returns a (1.) case NSM client.
//             - ctx    - context for the lifecycle of the *Client* itself.  Cancel when discarding the client.
//             - name   - name of the NetworkServiceMeshClient
//             - onHeal - *networkservice.NetworkServiceClient.  Since networkservice.NetworkServiceClient is an interface
//                        (and thus a pointer) *networkservice.NetworkServiceClient is a double pointer.  Meaning it
//                        points to a place that points to a place that implements networkservice.NetworkServiceClient
//                        This is done because when we use heal.NewClient as part of a chain, we may not *have*
//                        a pointer to this
//                        client used 'onHeal'.  If we detect we need to heal, onHeal.Request is used to heal.
//                        If onHeal is nil, then we simply set onHeal to this client chain element
//                        If we are part of a larger chain or a server, we should pass the resulting chain into
//                        this constructor before we actually have a pointer to it.
//                        If onHeal nil, onHeal will be pointed to the returned networkservice.NetworkServiceClient
//             - cc - grpc.ClientConnInterface for the endpoint to which this client should connect
//             - additionalFunctionality - any additional NetworkServiceClient chain elements to be included in the chain
func NewClient(ctx context.Context, name string, onHeal *networkservice.NetworkServiceClient, tokenGenerator token.GeneratorFunc, cc grpc.ClientConnInterface, additionalFunctionality ...networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	var rv networkservice.NetworkServiceClient
	if onHeal == nil {
		onHeal = &rv
	}
	rv = chain.NewNetworkServiceClient(
		append(
			append([]networkservice.NetworkServiceClient{
				authorize.NewClient(),
				updatepath.NewClient(name),
				serialize.NewClient(),
				heal.NewClient(ctx, networkservice.NewMonitorConnectionClient(cc), onHeal),
				refresh.NewClient(ctx),
				metadata.NewClient(),
			}, additionalFunctionality...),
			injectpeer.NewClient(),
			updatetoken.NewClient(tokenGenerator),
			networkservice.NewNetworkServiceClient(cc),
		)...)
	return rv
}

// NewCrossConnectClientFactory - returns a (2.) case func(cc grpc.ClientConnInterface) NSM client factory.
func NewCrossConnectClientFactory(name string, onHeal *networkservice.NetworkServiceClient, tokenGenerator token.GeneratorFunc, additionalFunctionality ...networkservice.NetworkServiceClient) func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
	return func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
		return chain.NewNetworkServiceClient(
			mechanismtranslation.NewClient(),
			NewClient(ctx, name, onHeal, tokenGenerator, cc, additionalFunctionality...),
		)
	}
}

// NewClientFactory - returns a (3.) case func(cc grpc.ClientConnInterface) NSM client factory.
func NewClientFactory(name string, onHeal *networkservice.NetworkServiceClient, tokenGenerator token.GeneratorFunc, additionalFunctionality ...networkservice.NetworkServiceClient) func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
	return func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
		return NewClient(ctx, name, onHeal, tokenGenerator, cc, additionalFunctionality...)
	}
}
