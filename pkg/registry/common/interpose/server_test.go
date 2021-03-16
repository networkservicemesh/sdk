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
	"net/url"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/interpose"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/stringurl"
)

const (
	nameSuffix = "#interpose-nse"
	name       = "nse"
	validURL   = "tcp://0.0.0.0"
)

func TestInterposeRegistryServer_Interpose(t *testing.T) {
	captureName := new(captureNameTestRegistryServer)

	var crossMap stringurl.Map
	server := next.NewNetworkServiceEndpointRegistryServer(
		interpose.NewNetworkServiceEndpointRegistryServer(&crossMap),
		captureName,
	)

	reg, err := server.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "nse" + nameSuffix,
		Url:  validURL,
	})
	require.NoError(t, err)

	require.True(t, interpose.Is(reg.Name))
	require.Empty(t, captureName.name)
	requireCrossMapEqual(t, map[string]string{reg.Name: validURL}, &crossMap)

	_, err = server.Unregister(context.Background(), reg)
	require.NoError(t, err)

	require.Empty(t, captureName.name)
	requireCrossMapEqual(t, map[string]string{}, &crossMap)
}

func TestInterposeRegistryServer_Common(t *testing.T) {
	captureName := new(captureNameTestRegistryServer)

	var crossMap stringurl.Map
	server := next.NewNetworkServiceEndpointRegistryServer(
		interpose.NewNetworkServiceEndpointRegistryServer(&crossMap),
		captureName,
	)

	reg, err := server.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: name,
	})
	require.NoError(t, err)

	require.Equal(t, name, reg.Name)
	require.Equal(t, name, captureName.name)
	requireCrossMapEqual(t, map[string]string{}, &crossMap)

	captureName.name = ""

	_, err = server.Unregister(context.Background(), reg)
	require.NoError(t, err)

	require.Equal(t, name, captureName.name)
	requireCrossMapEqual(t, map[string]string{}, &crossMap)
}

func TestInterposeRegistryServer_Invalid(t *testing.T) {
	captureName := new(captureNameTestRegistryServer)

	var crossMap stringurl.Map
	server := next.NewNetworkServiceEndpointRegistryServer(
		interpose.NewNetworkServiceEndpointRegistryServer(&crossMap),
		captureName,
	)

	_, err := server.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "nse" + nameSuffix,
	})
	require.Error(t, err)

	require.Empty(t, captureName.name)
	requireCrossMapEqual(t, map[string]string{}, &crossMap)
}

func requireCrossMapEqual(t *testing.T, expected map[string]string, crossMap *stringurl.Map) {
	actual := map[string]string{}
	crossMap.Range(func(key string, value *url.URL) bool {
		actual[key] = value.String()
		return true
	})
	require.Equal(t, expected, actual)
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
