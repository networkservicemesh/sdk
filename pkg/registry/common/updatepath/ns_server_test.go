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
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/inject/injectpeertoken"
)

type nsSample struct {
	name string
	test func(t *testing.T)
}

var nsSamples = []*nsSample{
	{
		name: "EmptyPathInRequest",
		test: func(t *testing.T) {
			t.Cleanup(func() { goleak.VerifyNone(t) })

			clientToken, _, _ := tokenGeneratorFunc(clientID)(nil)
			serverToken, _, _ := tokenGeneratorFunc(serverID)(nil)

			want := &grpcmetadata.Path{
				Index: 0,
				PathSegments: []*grpcmetadata.PathSegment{
					{Token: clientToken},
					{Token: serverToken},
				},
			}

			server := next.NewNetworkServiceRegistryServer(
				injectpeertoken.NewNetworkServiceRegistryServer(tokenGeneratorFunc(clientID)),
				updatepath.NewNetworkServiceRegistryServer(tokenGeneratorFunc(serverID)),
			)

			path := &grpcmetadata.Path{}
			ctx := grpcmetadata.PathWithContext(context.Background(), path)
			nse, err := server.Register(ctx, &registry.NetworkService{})
			// Note: Its up to authorization to decide that we won't accept requests without a Path from the client
			require.NoError(t, err)
			require.NotNil(t, nse)

			equalJSON(t, want, path)
			equalJSON(t, []string{clientID, serverID}, nse.GetPathIds())
		},
	},
	{
		name: "InvalidPathIndex",
		test: func(t *testing.T) {
			t.Cleanup(func() { goleak.VerifyNone(t) })
			nse := &registry.NetworkService{}
			server := updatepath.NewNetworkServiceRegistryServer(tokenGeneratorFunc(serverID))

			path := &grpcmetadata.Path{
				Index: 1,
				PathSegments: []*grpcmetadata.PathSegment{
					{Token: "token"},
				},
			}
			ctx := grpcmetadata.PathWithContext(context.Background(), path)

			nse, err := server.Register(ctx, nse)
			require.Error(t, err)
			require.Nil(t, nse)
		},
	},
	{
		name: "ServerChain",
		test: func(t *testing.T) {
			t.Cleanup(func() { goleak.VerifyNone(t) })

			clientToken, _, _ := tokenGeneratorFunc(clientID)(nil)
			proxyToken, _, _ := tokenGeneratorFunc(proxyID)(nil)
			serverToken, _, _ := tokenGeneratorFunc(serverID)(nil)

			want := &grpcmetadata.Path{
				Index: 0,
				PathSegments: []*grpcmetadata.PathSegment{
					{Token: clientToken},
					{Token: proxyToken},
					{Token: serverToken},
				},
			}

			server := next.NewNetworkServiceRegistryServer(
				injectpeertoken.NewNetworkServiceRegistryServer(tokenGeneratorFunc(clientID)),
				updatepath.NewNetworkServiceRegistryServer(tokenGeneratorFunc(proxyID)),
				injectpeertoken.NewNetworkServiceRegistryServer(tokenGeneratorFunc(proxyID)),
				updatepath.NewNetworkServiceRegistryServer(tokenGeneratorFunc(serverID)),
			)

			path := &grpcmetadata.Path{}
			ctx := grpcmetadata.PathWithContext(context.Background(), path)

			nse, err := server.Register(ctx, &registry.NetworkService{})
			require.NoError(t, err)
			require.NotNil(t, nse)

			equalJSON(t, want, path)
			equalJSON(t, []string{clientID, proxyID, serverID}, nse.GetPathIds())
		},
	},
	{
		name: "Refresh",
		test: func(t *testing.T) {
			t.Cleanup(func() { goleak.VerifyNone(t) })

			clientToken, _, _ := tokenGeneratorFunc(clientID)(nil)
			proxyToken, _, _ := tokenGeneratorFunc(proxyID)(nil)
			serverToken, _, _ := tokenGeneratorFunc(serverID)(nil)

			ns := &registry.NetworkService{
				PathIds: []string{"id1", "id2", "id3"},
			}
			path := &grpcmetadata.Path{
				Index: 0,
				PathSegments: []*grpcmetadata.PathSegment{
					{Token: "token1"},
					{Token: "token2"},
					{Token: "token3"},
				},
			}

			want := &grpcmetadata.Path{
				Index: 0,
				PathSegments: []*grpcmetadata.PathSegment{
					{Token: clientToken},
					{Token: proxyToken},
					{Token: serverToken},
				},
			}

			server := next.NewNetworkServiceRegistryServer(
				injectpeertoken.NewNetworkServiceRegistryServer(tokenGeneratorFunc(clientID)),
				updatepath.NewNetworkServiceRegistryServer(tokenGeneratorFunc(proxyID)),
				injectpeertoken.NewNetworkServiceRegistryServer(tokenGeneratorFunc(proxyID)),
				updatepath.NewNetworkServiceRegistryServer(tokenGeneratorFunc(serverID)),
			)

			ctx := grpcmetadata.PathWithContext(context.Background(), path)
			nse, err := server.Register(ctx, ns)
			require.NoError(t, err)
			require.NotNil(t, nse)

			equalJSON(t, want, path)
			equalJSON(t, []string{clientID, proxyID, serverID}, nse.GetPathIds())
		},
	},
}

func TestNSUpdatePathServer(t *testing.T) {
	for i := range nsSamples {
		sample := nsSamples[i]
		t.Run("TestNSServer_"+sample.name, sample.test)
	}
}
