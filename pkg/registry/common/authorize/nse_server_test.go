// Copyright (c) 2022-2024 Cisco and/or its affiliates.
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

package authorize_test

import (
	"context"
	"math/rand"
	"net/url"
	"testing"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/nanoid"

	"go.uber.org/goleak"
)

func TestNetworkServiceEndpointRegistryAuthorization(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	server := authorize.NewNetworkServiceEndpointRegistryServer(authorize.WithPolicies("etc/nsm/opa/registry/client_allowed.rego"))
	require.NotNil(t, server)

	nse := &registry.NetworkServiceEndpoint{Name: "nse"}
	path1 := getPath(t, spiffeid1)
	ctx1 := grpcmetadata.PathWithContext(context.Background(), path1)

	path2 := getPath(t, spiffeid2)
	ctx2 := grpcmetadata.PathWithContext(context.Background(), path2)

	nse.PathIds = []string{spiffeid1}
	_, err := server.Register(ctx1, nse)
	require.NoError(t, err)

	nse.PathIds = []string{spiffeid2}
	_, err = server.Register(ctx2, nse)
	require.Error(t, err)

	nse.PathIds = []string{spiffeid1}
	_, err = server.Register(ctx1, nse)
	require.NoError(t, err)

	nse.PathIds = []string{spiffeid2}
	_, err = server.Unregister(ctx2, nse)
	require.Error(t, err)

	nse.PathIds = []string{spiffeid1}
	_, err = server.Unregister(ctx1, nse)
	require.NoError(t, err)
}

type randomErrorNSEServer struct {
	errorChance float32
}

func NewNetworkServiceEndpointRegistryServer(errorChance float32) registry.NetworkServiceEndpointRegistryServer {
	return &randomErrorNSEServer{
		errorChance: errorChance,
	}
}

func (s *randomErrorNSEServer) Register(ctx context.Context, ns *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	//nolint
	val := rand.Float32()
	if val > s.errorChance {
		return nil, errors.New("random error")
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, ns)
}

func (s *randomErrorNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *randomErrorNSEServer) Unregister(ctx context.Context, ns *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, ns)
}

type closeNSEData struct {
	nse  *registry.NetworkServiceEndpoint
	path *grpcmetadata.Path
}

func TestNetworkServiceEndpointRegistryAuthorize_ResourcePathIdMapHaveNoLeaks(t *testing.T) {
	var authorizeMap genericsync.Map[string, []string]
	server := next.NewNetworkServiceEndpointRegistryServer(
		authorize.NewNetworkServiceEndpointRegistryServer(
			authorize.WithPolicies("etc/nsm/opa/registry/client_allowed.rego"),
			authorize.WithResourcePathIdsMap(&authorizeMap)),
		NewNetworkServiceEndpointRegistryServer(0.5),
	)

	// Make 1000 requests with random spiffe IDs
	count := 1000
	data := make([]closeNSEData, 0)
	for i := 0; i < count; i++ {
		nseName, err := nanoid.GenerateString(10, nanoid.WithAlphabet("abcdefghijklmnopqrstuvwxyz"))
		require.NoError(t, err)
		spiffeidPath, err := nanoid.GenerateString(10, nanoid.WithAlphabet("abcdefghijklmnopqrstuvwxyz"))
		require.NoError(t, err)

		u := &url.URL{Scheme: "spiffe", Host: "test.com", Path: spiffeidPath}
		spiffeid := u.String()

		nse := &registry.NetworkServiceEndpoint{Name: nseName}
		nse.PathIds = []string{spiffeid}

		path := getPath(t, spiffeid)
		ctx := grpcmetadata.PathWithContext(context.Background(), path)

		nse, err = server.Register(ctx, nse)
		if err == nil {
			data = append(data, closeNSEData{nse: nse, path: path})
		}
	}

	// Close the connections established in the previous loop
	for _, closeData := range data {
		ctx := grpcmetadata.PathWithContext(context.Background(), closeData.path)
		_, err := server.Unregister(ctx, closeData.nse)
		require.NoError(t, err)
	}
	mapLen := 0
	authorizeMap.Range(func(key string, value []string) bool {
		mapLen++
		return true
	})
	require.Equal(t, mapLen, 0)
}
