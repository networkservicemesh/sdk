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
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type nsServerSample struct {
	name string
	test func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceRegistryServer)
}

// var nsServerSamples = []*nsServerSample{
// 	{
// 		name: "NoPath",
// 		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceRegistryServer) {
// 			t.Cleanup(func() {
// 				goleak.VerifyNone(t)
// 			})

// 			server := newUpdatePathServer(nse1)

// 			ns, err := server.Register(context.Background(), registerNSRequest(nil))
// 			require.NoError(t, err)
// 			require.NotNil(t, ns)

// 			path := path(0, 1)
// 			requirePathEqual(t, path, ns.Path, 0)
// 		},
// 	},
// 	{
// 		name: "SameName",
// 		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceRegistryServer) {
// 			t.Cleanup(func() {
// 				goleak.VerifyNone(t)
// 			})

// 			server := newUpdatePathServer(nse2)

// 			ns, err := server.Register(context.Background(), registerNSRequest(path(1, 2)))
// 			require.NoError(t, err)
// 			require.NotNil(t, ns)

// 			requirePathEqual(t, path(1, 2), ns.Path)
// 		},
// 	},
// 	{
// 		name: "DifferentName",
// 		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceRegistryServer) {
// 			t.Cleanup(func() {
// 				goleak.VerifyNone(t)
// 			})

// 			server := newUpdatePathServer(nse3)

// 			ns, err := server.Register(context.Background(), registerNSRequest(path(1, 2)))
// 			require.NoError(t, err)
// 			requirePathEqual(t, path(1, 3), ns.Path, 2)
// 		},
// 	},
// 	{
// 		name: "InvalidIndex",
// 		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceRegistryServer) {
// 			t.Cleanup(func() {
// 				goleak.VerifyNone(t)
// 			})

// 			server := newUpdatePathServer(nse3)

// 			_, err := server.Register(context.Background(), registerNSRequest(path(3, 2)))
// 			require.Error(t, err)
// 		},
// 	},
// 	{
// 		name: "DifferentNextName",
// 		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceRegistryServer) {
// 			t.Cleanup(func() {
// 				goleak.VerifyNone(t)
// 			})

// 			var nsPath *registry.Path
// 			server := next.NewNetworkServiceRegistryServer(
// 				newUpdatePathServer(nse3),
// 				checknse.NewNetworkServiceRegistryServer(t, func(t *testing.T, ns *registry.NetworkService) {
// 					nsPath = ns.Path
// 					requirePathEqual(t, path(2, 3), nsPath, 2)
// 				}),
// 			)

// 			requestPath := path(1, 3)
// 			requestPath.PathSegments[2].Name = "different"
// 			ns, err := server.Register(context.Background(), registerNSRequest(requestPath))
// 			require.NoError(t, err)
// 			require.NotNil(t, ns)

// 			nsPath.Index = 1
// 			requirePathEqual(t, nsPath, ns.Path, 2)
// 		},
// 	},
// 	{
// 		name: "NoNextAvailable",
// 		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceRegistryServer) {
// 			t.Cleanup(func() {
// 				goleak.VerifyNone(t)
// 			})

// 			var nsPath *registry.Path
// 			server := next.NewNetworkServiceRegistryServer(
// 				newUpdatePathServer(nse3),
// 				checknse.NewNetworkServiceRegistryServer(t, func(t *testing.T, ns *registry.NetworkService) {
// 					nsPath = ns.Path
// 					requirePathEqual(t, path(2, 3), nsPath, 2)
// 				}),
// 			)

// 			ns, err := server.Register(context.Background(), registerNSRequest(path(1, 2)))
// 			require.NoError(t, err)
// 			require.NotNil(t, ns)

// 			nsPath.Index = 1
// 			requirePathEqual(t, nsPath, ns.Path, 2)
// 		},
// 	},
// 	{
// 		name: "SameNextName",
// 		test: func(t *testing.T, newUpdatePathServer func(name string) registry.NetworkServiceRegistryServer) {
// 			t.Cleanup(func() {
// 				goleak.VerifyNone(t)
// 			})

// 			server := next.NewNetworkServiceRegistryServer(
// 				newUpdatePathServer(nse3),
// 				checknse.NewNetworkServiceRegistryServer(t, func(t *testing.T, ns *registry.NetworkService) {
// 					requirePathEqual(t, path(2, 3), ns.Path)
// 				}),
// 			)

// 			ns, err := server.Register(context.Background(), registerNSRequest(path(1, 3)))
// 			require.NoError(t, err)
// 			require.NotNil(t, ns)

// 			requirePathEqual(t, path(1, 3), ns.Path)
// 		},
// 	},
// }

// func TestUpdatePathNSServer(t *testing.T) {
// 	for i := range nsServerSamples {
// 		sample := nsServerSamples[i]
// 		t.Run("TestNewServer_"+sample.name, func(t *testing.T) {
// 			sample.test(t, updatepath.NewNetworkServiceRegistryServer)
// 		})
// 	}
// }
