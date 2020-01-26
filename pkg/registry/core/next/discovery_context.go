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

package next

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

const (
	nextDiscoveryServerKey contextKeyType = "NextDiscoveryServer"
	nextDiscoveryClientKey contextKeyType = "NextDiscoveryClient"
)

type contextKeyType string

// withNextDiscoveryServer -
//    Wraps 'parent' in a new Context that has the DiscoveryServer registry.NetworkServiceDiscoveryServer to be called in the chain
//    Should only be set in CompositeEndpoint.Request/Close
func withNextDiscoveryServer(parent context.Context, next registry.NetworkServiceDiscoveryServer) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextDiscoveryServerKey, next)
}

// DiscoveryServer -
//   Returns the DiscoveryServer networkservice.NetworkServiceServer to be called in the chain from the context.Context
func DiscoveryServer(ctx context.Context) registry.NetworkServiceDiscoveryServer {
	if rv, ok := ctx.Value(nextDiscoveryServerKey).(registry.NetworkServiceDiscoveryServer); ok {
		return rv
	}
	return nil
}

// withNextDiscoveryClient -
//    Wraps 'parent' in a new Context that has the DiscoveryServer registry.NetworkServiceDiscoveryClient to be called in the chain
//    Should only be set in CompositeEndpoint.Request/Close
func withNextDiscoveryClient(parent context.Context, next registry.NetworkServiceDiscoveryClient) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextDiscoveryClientKey, next)
}

// DiscoveryClient -
//   Returns the DiscoveryClient registry.NetworkServiceDiscoveryClient to be called in the chain from the context.Context
func DiscoveryClient(ctx context.Context) registry.NetworkServiceDiscoveryClient {
	if rv, ok := ctx.Value(nextDiscoveryClientKey).(registry.NetworkServiceDiscoveryClient); ok {
		return rv
	}
	return nil
}
