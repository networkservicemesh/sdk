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
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checknse"
)

type nsClientSample struct {
	name string
	test func(t *testing.T, newUpdatePathClient func(name string) registry.NetworkServiceRegistryClient)
}

var nsClientSamples = []*nsClientSample{
	{
		name: "NoPath",
		test: func(t *testing.T, newUpdatePathClient func(name string) registry.NetworkServiceRegistryClient) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathClient(nse1)
			path := &registry.Path{}
			_, err := server.Register(grpcmetadata.PathWithContext(context.Background(), path), &registry.NetworkService{})
			require.NoError(t, err)

			expected := makePath(0, 1)
			requirePathEqual(t, path, expected, 0)
		},
	},
	{
		name: "SameName",
		test: func(t *testing.T, newUpdatePathClient func(name string) registry.NetworkServiceRegistryClient) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathClient(nse2)

			path := makePath(1, 2)
			_, err := server.Register(grpcmetadata.PathWithContext(context.Background(), path), &registry.NetworkService{})
			require.NoError(t, err)

			requirePathEqual(t, path, makePath(1, 2))
		},
	},
	{
		name: "DifferentName",
		test: func(t *testing.T, newUpdatePathClient func(name string) registry.NetworkServiceRegistryClient) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathClient(nse3)

			path := makePath(1, 2)
			_, err := server.Register(grpcmetadata.PathWithContext(context.Background(), path), &registry.NetworkService{})
			require.NoError(t, err)
			requirePathEqual(t, path, makePath(1, 3), 2)
		},
	},
	{
		name: "InvalidIndex",
		test: func(t *testing.T, newUpdatePathClient func(name string) registry.NetworkServiceRegistryClient) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathClient(nse3)

			path := makePath(3, 2)
			_, err := server.Register(grpcmetadata.PathWithContext(context.Background(), path), &registry.NetworkService{})
			require.Error(t, err)
		},
	},
	{
		name: "DifferentNextName",
		test: func(t *testing.T, newUpdatePathClient func(name string) registry.NetworkServiceRegistryClient) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			var nsPath *registry.Path
			server := next.NewNetworkServiceRegistryClient(
				newUpdatePathClient(nse3),
				checknse.NewNetworkServiceRegistryClient(t, func(t *testing.T, ctx context.Context, ns *registry.NetworkService) {
					nsPath, _ = grpcmetadata.PathFromContext(ctx)
					requirePathEqual(t, makePath(2, 3), nsPath, 2)
				}),
			)

			path := makePath(1, 3)
			path.PathSegments[2].Name = different
			ns, err := server.Register(grpcmetadata.PathWithContext(context.Background(), path), &registry.NetworkService{})
			require.NoError(t, err)
			require.NotNil(t, ns)

			nsPath.Index = 1
			requirePathEqual(t, path, nsPath, 2)
		},
	},
	{
		name: "NoNextAvailable",
		test: func(t *testing.T, newUpdatePathClient func(name string) registry.NetworkServiceRegistryClient) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			var nsPath *registry.Path
			server := next.NewNetworkServiceRegistryClient(
				newUpdatePathClient(nse3),
				checknse.NewNetworkServiceRegistryClient(t, func(t *testing.T, ctx context.Context, ns *registry.NetworkService) {
					nsPath, _ = grpcmetadata.PathFromContext(ctx)
					requirePathEqual(t, makePath(2, 3), nsPath, 2)
				}),
			)

			path := makePath(1, 2)
			ns, err := server.Register(grpcmetadata.PathWithContext(context.Background(), path), &registry.NetworkService{})
			require.NoError(t, err)
			require.NotNil(t, ns)

			nsPath.Index = 1
			requirePathEqual(t, path, nsPath, 2)
		},
	},
	{
		name: "SameNextName",
		test: func(t *testing.T, newUpdatePathClient func(name string) registry.NetworkServiceRegistryClient) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := next.NewNetworkServiceRegistryClient(
				newUpdatePathClient(nse3),
				checknse.NewNetworkServiceRegistryClient(t, func(t *testing.T, ctx context.Context, ns *registry.NetworkService) {
					path, err := grpcmetadata.PathFromContext(ctx)
					require.NoError(t, err)
					requirePathEqual(t, makePath(2, 3), path)
				}),
			)

			path := makePath(1, 3)
			ns, err := server.Register(grpcmetadata.PathWithContext(context.Background(), path), &registry.NetworkService{})
			require.NoError(t, err)
			require.NotNil(t, ns)

			requirePathEqual(t, path, makePath(1, 3))
		},
	},
}

func TestUpdatePathNSClient(t *testing.T) {
	for i := range nsClientSamples {
		sample := nsClientSamples[i]
		t.Run("TestNetworkServiceRegistryClient_"+sample.name, func(t *testing.T) {
			sample.test(t, updatepath.NewNetworkServiceRegistryClient)
		})
	}

	for i := range nsClientSamples {
		sample := nsClientSamples[i]
		t.Run("TestNetworkServiceRegistryServer_"+sample.name, func(t *testing.T) {
			sample.test(t, func(name string) registry.NetworkServiceRegistryClient {
				return adapters.NetworkServiceServerToClient(updatepath.NewNetworkServiceRegistryServer(name))
			})
		})
	}
}
