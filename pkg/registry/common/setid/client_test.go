// Copyright (c) 2021 Doc.ai and/or its affiliates.
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
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/checkid"
	"github.com/networkservicemesh/sdk/pkg/registry/common/setid"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

func TestSetIDClient_EmptyName(t *testing.T) {
	c := setid.NewNetworkServiceEndpointRegistryClient()

	nse := &registry.NetworkServiceEndpoint{
		NetworkServiceNames: []string{"ns-1", "ns-2"},
	}

	// 1. Register
	reg1, err := c.Register(context.Background(), nse.Clone())
	require.NoError(t, err)
	require.Contains(t, reg1.Name, nse.NetworkServiceNames[0])
	require.Contains(t, reg1.Name, nse.NetworkServiceNames[1])

	reg2, err := c.Register(context.Background(), nse.Clone())
	require.NoError(t, err)
	require.Contains(t, reg2.Name, nse.NetworkServiceNames[0])
	require.Contains(t, reg2.Name, nse.NetworkServiceNames[1])

	require.NotEqual(t, reg1.Name, reg2.Name)

	// 2. Refresh
	refresh, err := c.Register(context.Background(), reg1.Clone())
	require.NoError(t, err)
	require.Equal(t, reg1.Name, refresh.Name)

	// 3. Unregister
	_, err = c.Unregister(context.Background(), reg1.Clone())
	require.NoError(t, err)

	refresh, err = c.Register(context.Background(), reg1.Clone())
	require.NoError(t, err)

	require.NotEqual(t, reg1.Name, refresh.Name)
	require.NotEqual(t, reg2.Name, refresh.Name)
}

func TestSetIDClient_NotEmptyName(t *testing.T) {
	c := setid.NewNetworkServiceEndpointRegistryClient()

	nse := &registry.NetworkServiceEndpoint{
		Name: "nse",
	}

	reg1, err := c.Register(context.Background(), nse.Clone())
	require.NoError(t, err)
	require.Contains(t, reg1.Name, nse.Name)

	reg2, err := c.Register(context.Background(), nse.Clone())
	require.NoError(t, err)
	require.Contains(t, reg2.Name, nse.Name)

	require.NotEqual(t, reg1.Name, reg2.Name)
}

func TestSetIDClient_Duplicate(t *testing.T) {
	c := next.NewNetworkServiceEndpointRegistryClient(
		setid.NewNetworkServiceEndpointRegistryClient(),
		&errorClient{
			expected: 3,
			err:      new(checkid.DuplicateError),
		},
	)

	reg, err := c.Register(context.Background(), new(registry.NetworkServiceEndpoint))
	require.NoError(t, err)
	require.NotEmpty(t, reg.Name)
}

func TestSetIDClient_Error(t *testing.T) {
	c := next.NewNetworkServiceEndpointRegistryClient(
		setid.NewNetworkServiceEndpointRegistryClient(),
		&errorClient{
			expected: 1,
			err:      errors.New("error"),
		},
	)

	_, err := c.Register(context.Background(), new(registry.NetworkServiceEndpoint))
	require.Error(t, err)
}

type errorClient struct {
	count, expected int
	err             error

	registry.NetworkServiceEndpointRegistryClient
}

func (c *errorClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	defer func() { c.count++ }()

	if c.expected > c.count {
		return nil, c.err
	}

	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
}

func (c *errorClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}
