// Copyright (c) 2022 Cisco and/or its affiliates.
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

package updatepath_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checknse"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

type nseClientSample struct {
	name string
	test func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryClient)
}

var nseClientSamples = []*nseClientSample{
	{
		name: "NoPath",
		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryClient) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathServer(nse1)

			path := makePath(0, 1)
			_, err := server.Register(grpcmetadata.PathWithContext(context.Background(), path), &registry.NetworkServiceEndpoint{})
			require.NoError(t, err)
			requirePathEqual(t, path, makePath(0, 1), 0)
		},
	},
	{
		name: "SameName",
		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryClient) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathServer(nse2)

			path := makePath(1, 2)
			_, err := server.Register(grpcmetadata.PathWithContext(context.Background(), path), &registry.NetworkServiceEndpoint{})
			require.NoError(t, err)
			requirePathEqual(t, path, makePath(1, 2))
		},
	},
	{
		name: "DifferentName",
		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryClient) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathServer(nse3)

			path := makePath(1, 2)
			_, err := server.Register(grpcmetadata.PathWithContext(context.Background(), path), &registry.NetworkServiceEndpoint{})
			require.NoError(t, err)
			requirePathEqual(t, path, makePath(1, 3), 2)
		},
	},
	{
		name: "InvalidIndex",
		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryClient) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathServer(nse3)

			path := makePath(3, 2)
			_, err := server.Register(grpcmetadata.PathWithContext(context.Background(), path), &registry.NetworkServiceEndpoint{})
			require.Error(t, err)
		},
	},
	{
		name: "DifferentNextName",
		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryClient) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			var nsePath *registry.Path
			server := next.NewNetworkServiceEndpointRegistryClient(
				newUpdatePathServer(nse3),
				checknse.NewClient(t, func(t *testing.T, ctx context.Context, nse *registry.NetworkServiceEndpoint) {
					nsePath, _ = grpcmetadata.PathFromContext(ctx)
					requirePathEqual(t, makePath(2, 3), nsePath, 2)
				}),
			)

			path := makePath(1, 3)
			path.PathSegments[2].Name = "different"
			nse, err := server.Register(grpcmetadata.PathWithContext(context.Background(), path), &registry.NetworkServiceEndpoint{})
			require.NoError(t, err)
			require.NotNil(t, nse)

			nsePath.Index = 1
			requirePathEqual(t, path, nsePath, 2)
		},
	},
	{
		name: "NoNextAvailable",
		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryClient) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			var nsePath *registry.Path
			server := next.NewNetworkServiceEndpointRegistryClient(
				newUpdatePathServer(nse3),
				checknse.NewClient(t, func(t *testing.T, ctx context.Context, nse *registry.NetworkServiceEndpoint) {
					nsePath, _ = grpcmetadata.PathFromContext(ctx)
					requirePathEqual(t, makePath(2, 3), nsePath, 2)
				}),
			)

			path := makePath(1, 2)
			nse, err := server.Register(grpcmetadata.PathWithContext(context.Background(), path), &registry.NetworkServiceEndpoint{})
			require.NoError(t, err)
			require.NotNil(t, nse)

			nsePath.Index = 1
			requirePathEqual(t, path, nsePath, 2)
		},
	},
	{
		name: "SameNextName",
		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryClient) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := next.NewNetworkServiceEndpointRegistryClient(
				newUpdatePathServer(nse3),
				checknse.NewClient(t, func(t *testing.T, ctx context.Context, nse *registry.NetworkServiceEndpoint) {
					path, _ := grpcmetadata.PathFromContext(ctx)
					requirePathEqual(t, path, makePath(2, 3))
				}),
			)

			path := makePath(1, 3)
			nse, err := server.Register(grpcmetadata.PathWithContext(context.Background(), path), &registry.NetworkServiceEndpoint{})
			require.NoError(t, err)
			require.NotNil(t, nse)

			requirePathEqual(t, path, makePath(1, 3))
		},
	},
}

func TestUpdatePath(t *testing.T) {
	for i := range nseClientSamples {
		sample := nseClientSamples[i]
		t.Run("TestNetworkServiceEndpointRegistryClient_"+sample.name, func(t *testing.T) {
			sample.test(t, updatepath.NewNetworkServiceEndpointRegistryClient)
		})
	}

	for i := range nseClientSamples {
		sample := nseClientSamples[i]
		t.Run("TestNetworkServiceRegistryServer_"+sample.name, func(t *testing.T) {
			sample.test(t, func(name string) registry.NetworkServiceEndpointRegistryClient {
				return adapters.NetworkServiceEndpointServerToClient(updatepath.NewNetworkServiceEndpointRegistryServer(name))
			})
		})
	}
}
