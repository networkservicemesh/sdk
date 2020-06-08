// Copyright (c) 2020 Cisco Systems, Inc.
//
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

// Package next provides a mechanism for chained registry.{Registry,Discovery}{Server,Client}s to call
// the next element in the chain.
package next

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type contextKeyType string

const (
	nextDiscoveryServerKey   contextKeyType = "NextDiscoveryServer"
	nextDiscoveryClientKey   contextKeyType = "NextDiscoveryClient"
	nextRegistryServerKey    contextKeyType = "NextRegistryServer"
	nextRegistryClientKey    contextKeyType = "NextRegistryClient"
	nextNSMRegistryServerKey contextKeyType = "NextNSMRegistryServer"
	nextNSMRegistryClientKey contextKeyType = "NextNSMRegistryClient"
)

// withNextRegistryServer -
//    Wraps 'parent' in a new Context that has the DiscoveryServer registry.NetworkServiceRegistryServer to be called in the chain
func withNextRegistryServer(parent context.Context, next registry.NetworkServiceRegistryServer) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextRegistryServerKey, next)
}

// NetworkServiceRegistryServer -
//   Returns the NetworkServiceRegistryServer registry.NetworkServiceRegistryServer to be called in the chain from the context.Context
func NetworkServiceRegistryServer(ctx context.Context) registry.NetworkServiceRegistryServer {
	rv, ok := ctx.Value(nextRegistryServerKey).(registry.NetworkServiceRegistryServer)
	if !ok {
		client, ok := ctx.Value(nextRegistryClientKey).(registry.NetworkServiceRegistryClient)
		if ok {
			rv = adapters.NewRegistryClientToServer(client)
		}
	}
	if rv != nil {
		return rv
	}
	return &tailNetworkServiceRegistryServer{}
}

// withNextRegistryClient -
//    Wraps 'parent' in a new Context that has the NetworkServiceRegistryClient registry.NetworkServiceRegistryClientWrapper to be called in the chain
func withNextRegistryClient(parent context.Context, next registry.NetworkServiceRegistryClient) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextRegistryClientKey, next)
}

// NetworkServiceRegistryClient -
//   Returns the NetworkServiceRegistryClient registry.NetworkServiceRegistryClientWrapper to be called in the chain from the context.Context
func NetworkServiceRegistryClient(ctx context.Context) registry.NetworkServiceRegistryClient {
	rv, ok := ctx.Value(nextRegistryClientKey).(registry.NetworkServiceRegistryClient)
	if !ok {
		server, ok := ctx.Value(nextRegistryServerKey).(registry.NetworkServiceRegistryServer)
		if ok {
			rv = adapters.NewRegistryServerToClient(server)
		}
	}
	if rv != nil {
		return rv
	}
	return &tailNetworkServiceRegistryClient{}
}

// withNextDiscoveryServer -
//    Wraps 'parent' in a new Context that has the DiscoveryServer registry.NetworkServiceDiscoveryServer to be called in the chain
func withNextDiscoveryServer(parent context.Context, next registry.NetworkServiceDiscoveryServer) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextDiscoveryServerKey, next)
}

// DiscoveryServer -
//   Returns the DiscoveryServer networkservice.NetworkServiceServer to be called in the chain from the context.Context
func DiscoveryServer(ctx context.Context) registry.NetworkServiceDiscoveryServer {
	rv, ok := ctx.Value(nextDiscoveryServerKey).(registry.NetworkServiceDiscoveryServer)
	if !ok {
		client, ok := ctx.Value(nextDiscoveryClientKey).(registry.NetworkServiceDiscoveryClient)
		if ok {
			rv = adapters.NewDiscoveryClientToServer(client)
		}
	}
	if rv != nil {
		return rv
	}
	return &tailDiscoveryServer{}
}

// withNextDiscoveryClient -
//    Wraps 'parent' in a new Context that has the DiscoveryServer registry.NetworkServiceDiscoveryClient to be called in the chain
func withNextDiscoveryClient(parent context.Context, next registry.NetworkServiceDiscoveryClient) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextDiscoveryClientKey, next)
}

// DiscoveryClient -
//   Returns the DiscoveryClient registry.NetworkServiceDiscoveryClient to be called in the chain from the context.Context
func DiscoveryClient(ctx context.Context) registry.NetworkServiceDiscoveryClient {
	rv, ok := ctx.Value(nextDiscoveryClientKey).(registry.NetworkServiceDiscoveryClient)
	if !ok {
		server, ok := ctx.Value(nextDiscoveryServerKey).(registry.NetworkServiceDiscoveryServer)
		if ok {
			rv = adapters.NewDiscoveryServerToClient(server)
		}
	}
	if rv != nil {
		return rv
	}
	return &tailDiscoveryClient{}
}

// NSMRegistryClient -
//   Returns the NSMRegistryClient registry.NSMRegistryClient to be called in the chain from the context.Context
func NSMRegistryClient(ctx context.Context) registry.NsmRegistryClient {
	rv, ok := ctx.Value(nextNSMRegistryClientKey).(registry.NsmRegistryClient)
	if !ok {
		client, ok := ctx.Value(nextNSMRegistryServerKey).(registry.NsmRegistryServer)
		if ok {
			rv = adapters.NewNSMServerToClient(client)
		}
	}
	if rv != nil {
		return rv
	}
	return &tailRegistryNSMClient{}
}

// withNSMRegistryServer -
//    Wraps 'parent' in a new Context that has the NSMRegistryServer registry.NSMRegistryServer to be called in the chain
func withNSMRegistryServer(parent context.Context, next registry.NsmRegistryServer) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextNSMRegistryServerKey, next)
}

// NSMRegistryServer -
//   Returns the DiscoveryServer networkservice.NSMRegistryServer to be called in the chain from the context.Context
func NSMRegistryServer(ctx context.Context) registry.NsmRegistryServer {
	rv, ok := ctx.Value(nextNSMRegistryServerKey).(registry.NsmRegistryServer)
	if !ok {
		client, ok := ctx.Value(nextDiscoveryClientKey).(registry.NsmRegistryClient)
		if ok {
			rv = adapters.NewNSMClientToServer(client)
		}
	}
	if rv != nil {
		return rv
	}
	return &tailRegistryNSMServer{}
}

// withNSMRegistryClient -
//    Wraps 'parent' in a new Context that has the DiscoveryServer registry.NSMRegistryClient to be called in the chain
func withNSMRegistryClient(parent context.Context, next registry.NsmRegistryClient) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextNSMRegistryClientKey, next)
}
