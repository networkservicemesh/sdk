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

func TestNetworkServiceRegistryAuthorization(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	server := authorize.NewNetworkServiceRegistryServer(authorize.WithPolicies("etc/nsm/opa/registry/client_allowed.rego"))
	require.NotNil(t, server)

	ns := &registry.NetworkService{Name: "ns"}
	path1 := getPath(t, spiffeid1)
	ctx1 := grpcmetadata.PathWithContext(context.Background(), path1)

	path2 := getPath(t, spiffeid2)
	ctx2 := grpcmetadata.PathWithContext(context.Background(), path2)

	ns.PathIds = []string{spiffeid1}
	_, err := server.Register(ctx1, ns)
	require.NoError(t, err)

	ns.PathIds = []string{spiffeid2}
	_, err = server.Register(ctx2, ns)
	require.Error(t, err)

	ns.PathIds = []string{spiffeid1}
	_, err = server.Register(ctx1, ns)
	require.NoError(t, err)

	ns.PathIds = []string{spiffeid2}
	_, err = server.Unregister(ctx2, ns)
	require.Error(t, err)

	ns.PathIds = []string{spiffeid1}
	_, err = server.Unregister(ctx1, ns)
	require.NoError(t, err)
}

type randomErrorNSServer struct {
	errorChance float32
}

func NewNetworkServiceRegistryServer(errorChance float32) registry.NetworkServiceRegistryServer {
	return &randomErrorNSServer{
		errorChance: errorChance,
	}
}

func (s *randomErrorNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	//nolint
	val := rand.Float32()
	if val > s.errorChance {
		return nil, errors.New("random error")
	}
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
}

func (s *randomErrorNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (s *randomErrorNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}

type closeNSData struct {
	ns   *registry.NetworkService
	path *grpcmetadata.Path
}

func TestNetworkServiceRegistryAuthorize_ResourcePathIdMapHaveNoLeaks(t *testing.T) {
	var authorizeMap genericsync.Map[string, []string]
	server := next.NewNetworkServiceRegistryServer(
		authorize.NewNetworkServiceRegistryServer(
			authorize.WithPolicies("etc/nsm/opa/registry/client_allowed.rego"),
			authorize.WithResourcePathIdsMap(&authorizeMap)),
		NewNetworkServiceRegistryServer(0.5),
	)

	// Make 1000 requests with random spiffe IDs
	count := 1000
	data := make([]closeNSData, 0)
	for i := 0; i < count; i++ {
		nsName, err := nanoid.GenerateString(10, nanoid.WithAlphabet("abcdefghijklmnopqrstuvwxyz"))
		require.NoError(t, err)
		spiffeidPath, err := nanoid.GenerateString(10, nanoid.WithAlphabet("abcdefghijklmnopqrstuvwxyz"))
		require.NoError(t, err)

		u := &url.URL{Scheme: "spiffe", Host: "test.com", Path: spiffeidPath}
		spiffeid := u.String()

		ns := &registry.NetworkService{Name: nsName}
		ns.PathIds = []string{spiffeid}

		path := getPath(t, spiffeid)
		ctx := grpcmetadata.PathWithContext(context.Background(), path)

		ns, err = server.Register(ctx, ns)
		if err == nil {
			data = append(data, closeNSData{ns: ns, path: path})
		}
	}

	// Close the connections established in the previous loop
	for _, closeData := range data {
		ctx := grpcmetadata.PathWithContext(context.Background(), closeData.path)
		_, err := server.Unregister(ctx, closeData.ns)
		require.NoError(t, err)
	}
	mapLen := 0
	authorizeMap.Range(func(key string, value []string) bool {
		mapLen++
		return true
	})
	require.Equal(t, mapLen, 0)
}
