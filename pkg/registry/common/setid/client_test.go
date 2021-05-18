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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/checkid"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/setid"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

const (
	url1 = "tcp://1.1.1.1"
	url2 = "tcp://2.2.2.2"
)

func testEndpoints(nse *registry.NetworkServiceEndpoint) (nses [2]*registry.NetworkServiceEndpoint) {
	nses[0] = nse.Clone()
	nses[0].Url = url1

	nses[1] = nse.Clone()
	nses[1].Url = url2

	return nses
}

func TestSetIDClient(t *testing.T) {
	samples := []struct {
		name      string
		nse       *registry.NetworkServiceEndpoint
		nameCheck func(t *testing.T, name string)
	}{
		{
			name: "Empty name",
			nse: &registry.NetworkServiceEndpoint{
				NetworkServiceNames: []string{"ns-1", "ns-2"},
			},
			nameCheck: func(t *testing.T, name string) {
				require.Contains(t, name, "ns-1")
				require.Contains(t, name, "ns-2")
			},
		},
		{
			name: "Empty NS names",
			nse: &registry.NetworkServiceEndpoint{
				Name: "nse",
			},
			nameCheck: func(t *testing.T, name string) {
				require.Contains(t, name, "nse")
			},
		},
		{
			name: "All set",
			nse: &registry.NetworkServiceEndpoint{
				Name:                "nse",
				NetworkServiceNames: []string{"ns-1", "ns-2"},
			},
			nameCheck: func(t *testing.T, name string) {
				require.Contains(t, name, "nse")
			},
		},
	}

	for _, sample := range samples {
		// nolint:scopelint
		t.Run(sample.name, func(t *testing.T) {
			testSetIDClient(t, sample.nse, sample.nameCheck)
		})
	}
}

func testSetIDClient(t *testing.T, nse *registry.NetworkServiceEndpoint, nameCheck func(t *testing.T, name string)) {
	c := next.NewNetworkServiceEndpointRegistryClient(
		setid.NewNetworkServiceEndpointRegistryClient(),
		adapters.NetworkServiceEndpointServerToClient(next.NewNetworkServiceEndpointRegistryServer(
			checkid.NewNetworkServiceEndpointRegistryServer(),
			memory.NewNetworkServiceEndpointRegistryServer(),
		)),
	)

	nses := testEndpoints(nse)

	// 1. Register
	var regs [2]*registry.NetworkServiceEndpoint
	var err error
	for i := range nses {
		regs[i], err = c.Register(context.Background(), nses[i].Clone())
		require.NoError(t, err)
		nameCheck(t, regs[i].Name)
	}

	require.NotEqual(t, regs[0].Name, regs[1].Name)

	// 2. Refresh
	for _, reg := range regs {
		refresh, refreshErr := c.Register(context.Background(), reg.Clone())
		require.NoError(t, refreshErr)
		require.Equal(t, reg.Name, refresh.Name)
	}

	// 3. Unregister
	_, err = c.Unregister(context.Background(), regs[0].Clone())
	require.NoError(t, err)

	refresh, err := c.Register(context.Background(), regs[0].Clone())
	require.NoError(t, err)

	require.NotEqual(t, regs[0].Name, refresh.Name)
	require.NotEqual(t, regs[1].Name, refresh.Name)
}

func TestSetIDClient_Duplicate(t *testing.T) {
	c := next.NewNetworkServiceEndpointRegistryClient(
		setid.NewNetworkServiceEndpointRegistryClient(),
		&errorClient{
			expected: 3,
			err:      status.Error(codes.AlreadyExists, ""),
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

func TestSetIDClient_Restore(t *testing.T) {
	samples := []struct {
		name string
		nse  *registry.NetworkServiceEndpoint
	}{
		{
			name: "Empty name",
			nse: &registry.NetworkServiceEndpoint{
				NetworkServiceNames: []string{"ns-1", "ns-2"},
			},
		},
		{
			name: "Empty NS names",
			nse: &registry.NetworkServiceEndpoint{
				Name: "nse",
			},
		},
		{
			name: "All set",
			nse: &registry.NetworkServiceEndpoint{
				Name:                "nse",
				NetworkServiceNames: []string{"ns-1", "ns-2"},
			},
		},
	}

	for _, sample := range samples {
		// nolint:scopelint
		t.Run(sample.name, func(t *testing.T) {
			testSetIDClientRestore(t, sample.nse)
		})
	}
}

func testSetIDClientRestore(t *testing.T, nse *registry.NetworkServiceEndpoint) {
	s := next.NewNetworkServiceEndpointRegistryServer(
		checkid.NewNetworkServiceEndpointRegistryServer(),
		memory.NewNetworkServiceEndpointRegistryServer(),
	)

	nses := testEndpoints(nse)

	// 1. Register
	c := next.NewNetworkServiceEndpointRegistryClient(
		setid.NewNetworkServiceEndpointRegistryClient(),
		adapters.NetworkServiceEndpointServerToClient(s),
	)

	var regs [2]*registry.NetworkServiceEndpoint
	var err error
	for i := range nses {
		regs[i], err = c.Register(context.Background(), nses[i].Clone())
		require.NoError(t, err)
	}

	// 2. Restore registration
	c = next.NewNetworkServiceEndpointRegistryClient(
		setid.NewNetworkServiceEndpointRegistryClient(),
		adapters.NetworkServiceEndpointServerToClient(s),
	)

	var restores [2]*registry.NetworkServiceEndpoint
	for i := range nses {
		restores[i], err = c.Register(context.Background(), nses[i].Clone())
		require.NoError(t, err)
		require.Equal(t, regs[i].Name, restores[i].Name)
	}
}

type errorClient struct {
	count, expected int
	err             error
}

func (c *errorClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	defer func() { c.count++ }()

	if c.expected > c.count {
		return nil, c.err
	}

	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
}

func (c *errorClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *errorClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}
