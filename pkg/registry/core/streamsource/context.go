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

// Package streamsource provides a possibility to put stream source from the middle of the chain to the tail. Also, streasource allows use of multiple source chains.
package streamsource

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type contextKeyType string

const (
	nsStreamKey  contextKeyType = "NetworkServiceStream"
	nseStreamKey contextKeyType = "NetworkServiceEndpointStream"
)

// WithNetworkServiceStream puts NetworkServiceRegistry_FindClient to context. Putted stream will be used by streamsource.NetworkServiceRegistryClient on call Find
func WithNetworkServiceStream(ctx context.Context, stream registry.NetworkServiceRegistry_FindClient) context.Context {
	list, ok := ctx.Value(nsStreamKey).(*[]registry.NetworkServiceRegistry_FindClient)
	if !ok {
		list = &[]registry.NetworkServiceRegistry_FindClient{}
	}
	*list = append(*list, stream)
	return context.WithValue(ctx, nsStreamKey, list)
}

// WithNetworkServiceEndpointStream puts NetworkServiceEndpointRegistry_FindClient to context. Putted stream will be used by streamsource.NetworkServiceEndpointRegistry_FindClient on call Find
func WithNetworkServiceEndpointStream(ctx context.Context, stream registry.NetworkServiceEndpointRegistry_FindClient) context.Context {
	list, ok := ctx.Value(nseStreamKey).(*[]registry.NetworkServiceEndpointRegistry_FindClient)
	if !ok {
		list = &[]registry.NetworkServiceEndpointRegistry_FindClient{}
	}
	*list = append(*list, stream)
	return context.WithValue(ctx, nseStreamKey, list)
}

func networkServiceEndpointStreams(ctx context.Context) []registry.NetworkServiceEndpointRegistry_FindClient {
	list, ok := ctx.Value(nseStreamKey).(*[]registry.NetworkServiceEndpointRegistry_FindClient)
	if ok {
		return *list
	}
	return nil
}

func networkServiceStreams(ctx context.Context) []registry.NetworkServiceRegistry_FindClient {
	list, ok := ctx.Value(nsStreamKey).(*[]registry.NetworkServiceRegistry_FindClient)
	if ok {
		return *list
	}
	return nil
}
