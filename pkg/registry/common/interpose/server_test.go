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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/interpose"
	"github.com/networkservicemesh/sdk/pkg/tools/stringurl"
)

const (
	namePrefix = "interpose-nse#"
	name       = "nse"
	validURL   = "tcp://0.0.0.0"
)

func testServer() (*stringurl.Map, registry.NetworkServiceEndpointRegistryServer) {
	var crossMap stringurl.Map
	server := interpose.NewNetworkServiceRegistryServer(&crossMap)
	return &crossMap, server
}

func testNSE(name, u string) *registry.NetworkServiceEndpoint {
	return &registry.NetworkServiceEndpoint{
		Name: name,
		Url:  u,
	}
}

func TestInterposeRegistryServer_InterposeNSE(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	crossMap, server := testServer()

	reg1, err := server.Register(context.Background(), testNSE(namePrefix+name, validURL))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(reg1.Name, namePrefix))

	reg2, err := server.Register(context.Background(), testNSE(namePrefix+name, validURL))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(reg2.Name, namePrefix))

	require.NotEqual(t, reg1.Name, reg2.Name)

	requireCrossMapEqual(t, map[string]string{
		reg1.Name: validURL,
		reg2.Name: validURL,
	}, crossMap)

	_, err = server.Unregister(context.Background(), reg1)
	require.NoError(t, err)

	_, err = server.Unregister(context.Background(), reg2)
	require.NoError(t, err)

	requireCrossMapEqual(t, map[string]string{}, crossMap)
}

func TestInterposeRegistryServer_CommonNSE(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	crossMap, server := testServer()

	reg, err := server.Register(context.Background(), testNSE(name, ""))
	require.NoError(t, err)
	require.Equal(t, name, reg.Name)

	requireCrossMapEqual(t, map[string]string{}, crossMap)

	_, err = server.Unregister(context.Background(), reg.Clone())
	require.NoError(t, err)
}

func TestInterposeRegistryServer_InvalidNSE(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	crossMap, server := testServer()

	_, err := server.Register(context.Background(), testNSE(namePrefix+name, ""))
	require.Error(t, err)

	requireCrossMapEqual(t, map[string]string{}, crossMap)
}

func TestInterposeRegistryServer_RefreshInterposeNSE(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	crossMap, server := testServer()

	reg, err := server.Register(context.Background(), testNSE(namePrefix+name, validURL))
	require.NoError(t, err)

	requireCrossMapEqual(t, map[string]string{
		reg.Name: validURL,
	}, crossMap)

	name := reg.Name

	reg, err = server.Register(context.Background(), reg)
	require.NoError(t, err)
	require.Equal(t, name, reg.Name)

	requireCrossMapEqual(t, map[string]string{
		reg.Name: validURL,
	}, crossMap)

	_, err = server.Unregister(context.Background(), reg.Clone())
	require.NoError(t, err)

	requireCrossMapEqual(t, map[string]string{}, crossMap)
}

func requireCrossMapEqual(t *testing.T, expected map[string]string, crossMap *stringurl.Map) {
	actual := map[string]string{}
	crossMap.Range(func(key string, value *url.URL) bool {
		actual[key] = value.String()
		return true
	})
	require.Equal(t, expected, actual)
}
