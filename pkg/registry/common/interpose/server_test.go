// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package interpose_test

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/interpose"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

const (
	nameSuffix = "#interpose-nse"
	name       = "nse"
	validURL   = "tcp://0.0.0.0"
)

func requireReadList(t *testing.T, server registry.NetworkServiceEndpointRegistryServer) []*registry.NetworkServiceEndpoint {
	s, err := chain.NewNetworkServiceEndpointRegistryClient(
		interpose.NewNetworkServiceEndpointRegistryClient(),
		adapters.NetworkServiceEndpointServerToClient(server)).Find(context.Background(), &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{}})
	require.NoError(t, err)

	return registry.ReadNetworkServiceEndpointList(s)
}

func TestInterposeRegistryServer_Interpose(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	captureName := new(captureNameTestRegistryServer)

	server := next.NewNetworkServiceEndpointRegistryServer(
		interpose.NewNetworkServiceEndpointRegistryServer(),
		captureName,
	)

	reg, err := server.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: name + nameSuffix,
		Url:  validURL,
	})
	require.NoError(t, err)

	require.True(t, interpose.Is(reg.Name))
	require.Empty(t, captureName.name)

	var list = requireReadList(t, server)

	require.Len(t, list, 1)
	require.Equal(t, validURL, list[0].Url)

	_, err = server.Unregister(context.Background(), reg)
	require.NoError(t, err)

	require.Empty(t, requireReadList(t, server))
}

func TestInterposeRegistryServer_Common(t *testing.T) {
	captureName := new(captureNameTestRegistryServer)

	server := next.NewNetworkServiceEndpointRegistryServer(
		interpose.NewNetworkServiceEndpointRegistryServer(),
		captureName,
	)

	reg, err := server.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: name,
	})
	require.NoError(t, err)

	require.Equal(t, name, reg.Name)
	require.Equal(t, name, captureName.name)

	require.Empty(t, requireReadList(t, server), 1)

	captureName.name = ""

	_, err = server.Unregister(context.Background(), reg)
	require.NoError(t, err)

	require.Equal(t, name, captureName.name)

	require.Empty(t, requireReadList(t, server), 1)
}

func TestInterposeRegistryServer_Invalid(t *testing.T) {
	captureName := new(captureNameTestRegistryServer)

	server := next.NewNetworkServiceEndpointRegistryServer(
		interpose.NewNetworkServiceEndpointRegistryServer(),
		captureName,
	)

	_, err := server.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: name + nameSuffix,
	})
	require.Error(t, err)

	require.Empty(t, captureName.name)
	require.Empty(t, requireReadList(t, server), 1)
}

type captureNameTestRegistryServer struct {
	name string

	registry.NetworkServiceEndpointRegistryServer
}

func (r *captureNameTestRegistryServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	r.name = in.Name

	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (r *captureNameTestRegistryServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	r.name = in.Name

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}
func (r *captureNameTestRegistryServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	r.name = query.NetworkServiceEndpoint.GetName()
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}
