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
	"strings"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/sdk/pkg/registry/common/setid"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
)

const (
	nseName      = "nse-1"
	domain       = "domain.com"
	remoteSuffix = "-remote"
)

func testNSE(name string) *registry.NetworkServiceEndpoint {
	return &registry.NetworkServiceEndpoint{
		Name:                name,
		NetworkServiceNames: []string{"ns-1"},
	}
}

func TestSetIDServer_NewNSE(t *testing.T) {
	server := setid.NewNetworkServiceEndpointRegistryServer()

	reg1, err := server.Register(context.Background(), testNSE(nseName))
	require.NoError(t, err)

	reg2, err := server.Register(context.Background(), testNSE(nseName))
	require.NoError(t, err)

	require.NotEqual(t, reg1.Name, reg2.Name)

	_, err = server.Unregister(context.Background(), reg1)
	require.NoError(t, err)

	_, err = server.Unregister(context.Background(), reg2)
	require.NoError(t, err)
}

func TestSetIDServer_RefreshNSE(t *testing.T) {
	server := setid.NewNetworkServiceEndpointRegistryServer()

	reg, err := server.Register(context.Background(), testNSE(nseName))
	require.NoError(t, err)

	name := reg.Name

	reg, err = server.Register(context.Background(), reg)
	require.NoError(t, err)

	require.Equal(t, name, reg.Name)

	_, err = server.Unregister(context.Background(), reg)
	require.NoError(t, err)
}

func TestSetIDServer_InterDomainNSE(t *testing.T) {
	server := setid.NewNetworkServiceEndpointRegistryServer()

	reg, err := server.Register(context.Background(), testNSE(nseName+"@"+domain))
	require.NoError(t, err)

	require.True(t, strings.HasSuffix(reg.Name, "@"+domain))

	_, err = server.Unregister(context.Background(), reg)
	require.NoError(t, err)
}

func TestSetIDServer_RemoteRegistry(t *testing.T) {
	server := next.NewNetworkServiceEndpointRegistryServer(
		setid.NewNetworkServiceEndpointRegistryServer(),
		new(remoteRegistry),
	)

	reg, err := server.Register(context.Background(), testNSE(nseName))
	require.NoError(t, err)

	name := reg.Name

	reg, err = server.Register(context.Background(), reg)
	require.NoError(t, err)

	require.Equal(t, name, reg.Name)

	_, err = server.Unregister(context.Background(), reg)
	require.NoError(t, err)
}

type remoteRegistry struct {
	registry.NetworkServiceEndpointRegistryServer
}

func (s *remoteRegistry) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if !strings.HasSuffix(nse.Name, remoteSuffix) {
		nse.Name += remoteSuffix
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *remoteRegistry) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
