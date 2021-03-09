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

package setid_test

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"

	"github.com/networkservicemesh/sdk/pkg/registry/common/setid"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
)

func testNSE() *registry.NetworkServiceEndpoint {
	return &registry.NetworkServiceEndpoint{
		Name:                "nse-1",
		NetworkServiceNames: []string{"ns-1"},
	}
}

func TestSetIDServer_NewNSE(t *testing.T) {
	server := setid.NewNetworkServiceEndpointRegistryServer()

	reg1, err := server.Register(context.Background(), testNSE())
	require.NoError(t, err)

	reg2, err := server.Register(context.Background(), testNSE())
	require.NoError(t, err)

	require.NotEqual(t, reg1.Name, reg2.Name)

	_, err = server.Unregister(context.Background(), reg1)
	require.NoError(t, err)

	_, err = server.Unregister(context.Background(), reg2)
	require.NoError(t, err)
}

func TestSetIDServer_RefreshNSE(t *testing.T) {
	server := setid.NewNetworkServiceEndpointRegistryServer()

	reg, err := server.Register(context.Background(), testNSE())
	require.NoError(t, err)

	name := reg.Name

	reg, err = server.Register(context.Background(), reg)
	require.NoError(t, err)

	require.Equal(t, name, reg.Name)

	_, err = server.Unregister(context.Background(), reg)
	require.NoError(t, err)
}

func TestSetIDServer_RemoteRegistry(t *testing.T) {
	captureName := new(captureNameRegistryServer)

	server := next.NewNetworkServiceEndpointRegistryServer(
		setid.NewNetworkServiceEndpointRegistryServer(),
		setid.NewNetworkServiceEndpointRegistryServer(),
		setid.NewNetworkServiceEndpointRegistryServer(),
		setid.NewNetworkServiceEndpointRegistryServer(),
		captureName,
		setid.NewNetworkServiceEndpointRegistryServer(),
	)

	reg, err := server.Register(context.Background(), testNSE())
	require.NoError(t, err)

	name := reg.Name
	require.Less(t, len(name), 2*len(uuid.New().String()))
	require.Equal(t, captureName.name, name)

	reg, err = server.Register(context.Background(), reg)
	require.NoError(t, err)

	require.Equal(t, name, reg.Name)
	require.Equal(t, name, captureName.name)

	_, err = server.Unregister(context.Background(), reg)
	require.NoError(t, err)
}

type captureNameRegistryServer struct {
	name string

	registry.NetworkServiceEndpointRegistryServer
}

func (s *captureNameRegistryServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	reg, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}

	s.name = reg.Name

	return reg, nil
}

func (s *captureNameRegistryServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
