// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2022 Cisco Systems, Inc.
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

// Package next provides a mechanism for chained registry.{Registry,Discovery}{Server,Client}s to call
// the next element in the chain.
package next

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type contextKeyType string

const (
	nextNSRegistryServerKey  contextKeyType = "NextNSRegistryServer"
	nextNSRegistryClientKey  contextKeyType = "NextNSRegistryClient"
	nextNSERegistryServerKey contextKeyType = "NextNSERegistryServer"
	nextNSERegistryClientKey contextKeyType = "NextNSERegistryClient"
)

// withNextNSRegistryServer -
//
//	Wraps 'parent' in a new Context that has the DiscoveryServer registry.NetworkServiceRegistryServer to be called in the chain
func withNextNSRegistryServer(parent context.Context, next registry.NetworkServiceRegistryServer) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, nextNSRegistryServerKey, next)
}

// withNextNSRegistryClient -
//
//	Wraps 'parent' in a new Context that has the NetworkServiceRegistryClient registry.NetworkServiceRegistryClientWrapper to be called in the chain
func withNextNSRegistryClient(parent context.Context, next registry.NetworkServiceRegistryClient) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, nextNSRegistryClientKey, next)
}

// NetworkServiceRegistryServer -
//
//	Returns the NetworkServiceRegistryServer registry.NetworkServiceRegistryServer to be called in the chain from the context.Context
func NetworkServiceRegistryServer(ctx context.Context) registry.NetworkServiceRegistryServer {
	rv, ok := ctx.Value(nextNSRegistryServerKey).(registry.NetworkServiceRegistryServer)
	if ok && rv != nil {
		return rv
	}
	return &tailNetworkServiceRegistryServer{}
}

// NetworkServiceRegistryClient -
//
//	Returns the NetworkServiceRegistryClient registry.NetworkServiceRegistryClientWrapper to be called in the chain from the context.Context
func NetworkServiceRegistryClient(ctx context.Context) registry.NetworkServiceRegistryClient {
	rv, ok := ctx.Value(nextNSRegistryClientKey).(registry.NetworkServiceRegistryClient)
	if ok && rv != nil {
		return rv
	}
	return &tailNetworkServiceRegistryClient{}
}

// withNextNSERegistryServer -
//
//	Wraps 'parent' in a new Context that has the DiscoveryServer registry.NetworkServiceEndpointRegistryServer to be called in the chain
func withNextNSERegistryServer(parent context.Context, next registry.NetworkServiceEndpointRegistryServer) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, nextNSERegistryServerKey, next)
}

// withNextNSERegistryClient -
//
//	Wraps 'parent' in a new Context that has the NetworkServiceEndpointRegistryClient registry.NetworkServiceEndpointRegistryClientWrapper to be called in the chain
func withNextNSERegistryClient(parent context.Context, next registry.NetworkServiceEndpointRegistryClient) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, nextNSERegistryClientKey, next)
}

// NetworkServiceEndpointRegistryServer -
//
//	Returns the NetworkServiceEndpointRegistryServer registry.NetworkServiceEndpointRegistryServer to be called in the chain from the context.Context
func NetworkServiceEndpointRegistryServer(ctx context.Context) registry.NetworkServiceEndpointRegistryServer {
	rv, ok := ctx.Value(nextNSERegistryServerKey).(registry.NetworkServiceEndpointRegistryServer)
	if ok && rv != nil {
		return rv
	}
	return &tailNetworkServiceEndpointRegistryServer{}
}

// NetworkServiceEndpointRegistryClient -
//
//	Returns the NetworkServiceEndpointRegistryClient registry.NetworkServiceEndpointRegistryClientWrapper to be called in the chain from the context.Context
func NetworkServiceEndpointRegistryClient(ctx context.Context) registry.NetworkServiceEndpointRegistryClient {
	rv, ok := ctx.Value(nextNSERegistryClientKey).(registry.NetworkServiceEndpointRegistryClient)
	if ok && rv != nil {
		return rv
	}
	return &tailNetworkServiceEndpointRegistryClient{}
}
