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

	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checknse"
)

const (
	nse1           = "nse-1"
	nse2           = "nse-2"
	nse3           = "nse-3"
	pathSegmentID1 = "36ce7f0c-9f6d-40a4-8b39-6b56ff07eea9"
	pathSegmentID2 = "ece490ea-dfe8-4512-a3ca-5be7b39515c5"
	pathSegmentID3 = "f9a83e55-0a4f-3647-144a-98a9ee8fb231"
)

func registerRequest(path *registry.Path) *registry.NetworkServiceEndpoint {
	return &registry.NetworkServiceEndpoint{
		Name: "nse",
		Path: path,
	}
}

func path(pathIndex uint32, pathSegments int) *registry.Path {
	if pathSegments == 0 {
		return nil
	}

	path := &registry.Path{
		Index: pathIndex,
	}
	if pathSegments >= 1 {
		path.PathSegments = append(path.PathSegments, &registry.PathSegment{
			Name: nse1,
			Id:   pathSegmentID1,
		})
	}
	if pathSegments >= 2 {
		path.PathSegments = append(path.PathSegments, &registry.PathSegment{
			Name: nse2,
			Id:   pathSegmentID2,
		})
	}
	if pathSegments >= 3 {
		path.PathSegments = append(path.PathSegments, &registry.PathSegment{
			Name: nse3,
			Id:   pathSegmentID3,
		})
	}
	return path
}

func requirePathEqual(t *testing.T, expected, actual *registry.Path, unknownIDs ...int) {
	expected = expected.Clone()
	actual = actual.Clone()
	for _, index := range unknownIDs {
		expected.PathSegments[index].Id = ""
		actual.PathSegments[index].Id = ""
	}
	require.Equal(t, expected.String(), actual.String())
}

type sample struct {
	name string
	test func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryServer)
}

var samples = []*sample{
	{
		name: "NoPath",
		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathServer(nse1)

			nse, err := server.Register(context.Background(), registerRequest(nil))
			require.NoError(t, err)
			require.NotNil(t, nse)

			path := path(0, 1)
			requirePathEqual(t, path, nse.Path, 0)
		},
	},
	{
		name: "SameName",
		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathServer(nse2)

			nse, err := server.Register(context.Background(), registerRequest(path(1, 2)))
			require.NoError(t, err)
			require.NotNil(t, nse)

			requirePathEqual(t, path(1, 2), nse.Path)
		},
	},
	{
		name: "DifferentName",
		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathServer(nse3)

			nse, err := server.Register(context.Background(), registerRequest(path(1, 2)))
			require.NoError(t, err)
			requirePathEqual(t, path(1, 3), nse.Path, 2)
		},
	},
	{
		name: "InvalidIndex",
		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathServer(nse3)

			_, err := server.Register(context.Background(), registerRequest(path(3, 2)))
			require.Error(t, err)
		},
	},
	{
		name: "DifferentNextName",
		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			var nsePath *registry.Path
			server := next.NewNetworkServiceEndpointRegistryServer(
				newUpdatePathServer(nse3),
				checknse.NewServer(t, func(t *testing.T, nse *registry.NetworkServiceEndpoint) {
					nsePath = nse.Path
					requirePathEqual(t, path(2, 3), nsePath, 2)
				}),
			)

			requestPath := path(1, 3)
			requestPath.PathSegments[2].Name = "different"
			nse, err := server.Register(context.Background(), registerRequest(requestPath))
			require.NoError(t, err)
			require.NotNil(t, nse)

			nsePath.Index = 1
			requirePathEqual(t, nsePath, nse.Path, 2)
		},
	},
	{
		name: "NoNextAvailable",
		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			var nsePath *registry.Path
			server := next.NewNetworkServiceEndpointRegistryServer(
				newUpdatePathServer(nse3),
				checknse.NewServer(t, func(t *testing.T, nse *registry.NetworkServiceEndpoint) {
					nsePath = nse.Path
					requirePathEqual(t, path(2, 3), nsePath, 2)
				}),
			)

			nse, err := server.Register(context.Background(), registerRequest(path(1, 2)))
			require.NoError(t, err)
			require.NotNil(t, nse)

			nsePath.Index = 1
			requirePathEqual(t, nsePath, nse.Path, 2)
		},
	},
	{
		name: "SameNextName",
		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceEndpointRegistryServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := next.NewNetworkServiceEndpointRegistryServer(
				newUpdatePathServer(nse3),
				checknse.NewServer(t, func(t *testing.T, nse *registry.NetworkServiceEndpoint) {
					requirePathEqual(t, path(2, 3), nse.Path)
				}),
			)

			nse, err := server.Register(context.Background(), registerRequest(path(1, 3)))
			require.NoError(t, err)
			require.NotNil(t, nse)

			requirePathEqual(t, path(1, 3), nse.Path)
		},
	},
}

func TestUpdatePath(t *testing.T) {
	for i := range samples {
		sample := samples[i]
		t.Run("TestNewServer_"+sample.name, func(t *testing.T) {
			sample.test(t, updatepath.NewNetworkServiceEndpointRegistryServer)
		})
	}
}
