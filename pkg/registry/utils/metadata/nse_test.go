// Copyright (c) 2023 Cisco and/or its affiliates.
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

package metadata_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/metadata"
)

const (
	testNSEKey = "test"
)

func testNSE() *registry.NetworkServiceEndpoint {
	return &registry.NetworkServiceEndpoint{
		Name:                "nse",
		NetworkServiceNames: []string{"ns-name"},
	}
}

type nseSample struct {
	name string
	test func(t *testing.T, server registry.NetworkServiceEndpointRegistryServer, isClient bool)
}

var nseSamples = []*nseSample{
	{
		name: "Register",
		test: func(t *testing.T, server registry.NetworkServiceEndpointRegistryServer, isClient bool) {
			var actual, expected map[string]string = nil, map[string]string{"a": "A"}

			chainServer := next.NewNetworkServiceEndpointRegistryServer(
				server,
				checkcontext.NewNSEServer(t, func(_ *testing.T, ctx context.Context) {
					metadata.Map(ctx, isClient).Store(testNSEKey, expected)
				}),
			)
			_, err := chainServer.Register(context.Background(), testNSE())
			require.NoError(t, err)

			chainServer = next.NewNetworkServiceEndpointRegistryServer(
				server,
				checkcontext.NewNSEServer(t, func(_ *testing.T, ctx context.Context) {
					if raw, ok := metadata.Map(ctx, isClient).Load(testNSEKey); ok {
						actual = raw.(map[string]string)
					} else {
						actual = nil
					}
				}),
			)
			_, err = chainServer.Register(context.Background(), testNSE())
			require.NoError(t, err)

			require.Equal(t, expected, actual)
		},
	},
	{
		name: "Register failed",
		test: func(t *testing.T, server registry.NetworkServiceEndpointRegistryServer, isClient bool) {
			chainServer := next.NewNetworkServiceEndpointRegistryServer(
				server,
				checkcontext.NewNSEServer(t, func(_ *testing.T, ctx context.Context) {
					metadata.Map(ctx, isClient).Store(testNSEKey, 0)
				}),
				injecterror.NewNetworkServiceEndpointRegistryServer(),
			)
			_, err := chainServer.Register(context.Background(), testNSE())
			require.Error(t, err)

			chainServer = next.NewNetworkServiceEndpointRegistryServer(
				server,
				checkcontext.NewNSEServer(t, func(t *testing.T, ctx context.Context) {
					_, ok := metadata.Map(ctx, isClient).Load(testNSEKey)
					require.False(t, ok)
				}),
			)
			_, err = chainServer.Register(context.Background(), testNSE())
			require.NoError(t, err)
		},
	},
	{
		name: "Refresh failed",
		test: func(t *testing.T, server registry.NetworkServiceEndpointRegistryServer, isClient bool) {
			chainServer := next.NewNetworkServiceEndpointRegistryServer(
				server,
				checkcontext.NewNSEServer(t, func(_ *testing.T, ctx context.Context) {
					metadata.Map(ctx, isClient).Store(testNSEKey, 0)
				}),
				injecterror.NewNetworkServiceEndpointRegistryServer(
					injecterror.WithRegisterErrorTimes(1),
				),
			)
			_, err := chainServer.Register(context.Background(), testNSE())
			require.NoError(t, err)

			_, err = chainServer.Register(context.Background(), testNSE())
			require.Error(t, err)

			chainServer = next.NewNetworkServiceEndpointRegistryServer(
				server,
				checkcontext.NewNSEServer(t, func(t *testing.T, ctx context.Context) {
					_, ok := metadata.Map(ctx, isClient).Load(testNSEKey)
					require.True(t, ok)
				}),
			)
			_, err = chainServer.Register(context.Background(), testNSE())
			require.NoError(t, err)
		},
	},
	{
		name: "Unregister",
		test: func(t *testing.T, server registry.NetworkServiceEndpointRegistryServer, isClient bool) {
			data := map[string]string{"a": "A"}

			chainServer := next.NewNetworkServiceEndpointRegistryServer(
				server,
				checkcontext.NewNSEServer(t, func(_ *testing.T, ctx context.Context) {
					metadata.Map(ctx, isClient).Store(testNSEKey, data)
				}),
			)
			registeredNSE, err := chainServer.Register(context.Background(), testNSE())
			require.NoError(t, err)

			_, err = server.Unregister(context.Background(), registeredNSE)
			require.NoError(t, err)

			chainServer = next.NewNetworkServiceEndpointRegistryServer(
				server,
				checkcontext.NewNSEServer(t, func(_ *testing.T, ctx context.Context) {
					if raw, ok := metadata.Map(ctx, isClient).Load(testNSEKey); ok {
						data = raw.(map[string]string)
					} else {
						data = nil
					}
				}),
			)
			_, err = chainServer.Register(context.Background(), testNSE())
			require.NoError(t, err)

			require.Nil(t, data)
		},
	},
	{
		name: "Double Unregister",
		test: func(t *testing.T, server registry.NetworkServiceEndpointRegistryServer, isClient bool) {
			chainServer := next.NewNetworkServiceEndpointRegistryServer(
				server,
				checkcontext.NewNSEServer(t, func(t *testing.T, ctx context.Context) {
					require.NotNil(t, metadata.Map(ctx, isClient))
				}),
			)

			registeredNSE, err := chainServer.Register(context.Background(), testNSE())
			require.NoError(t, err)

			_, err = chainServer.Unregister(context.Background(), registeredNSE)
			require.NoError(t, err)

			_, err = chainServer.Unregister(context.Background(), registeredNSE)
			require.NoError(t, err)
		},
	},
}

func TestMetaDataNSEServer(t *testing.T) {
	for i := range nseSamples {
		sample := nseSamples[i]
		t.Run(sample.name, func(t *testing.T) {
			sample.test(t, metadata.NewNetworkServiceEndpointServer(), false)
		})
	}
}

func TestMetaDataNSEClient(t *testing.T) {
	for i := range nseSamples {
		sample := nseSamples[i]
		t.Run(sample.name, func(t *testing.T) {
			sample.test(t, adapters.NetworkServiceEndpointClientToServer(metadata.NewNetworkServiceEndpointClient()), true)
		})
	}
}
