// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package interpose_test define a tests for cross connect NSE chain element
package interpose_test

import (
	"context"
	"sync/atomic"
	"testing"

	adapters2 "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	next_reg "github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/networkservicemesh/sdk/pkg/registry/common/interpose"
	interpose_tools "github.com/networkservicemesh/sdk/pkg/tools/interpose"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
)

type crossNSEMap struct {
	endpoints interpose_tools.Map
	count     int64
}

func (c *crossNSEMap) LoadOrStore(name string, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, bool) {
	val, loaded := c.endpoints.LoadOrStore(name, request)
	if !loaded {
		atomic.AddInt64(&c.count, 1)
	}
	return val, loaded
}

func (c *crossNSEMap) Delete(name string) {
	c.endpoints.Delete(name)
}

func TestCrossNSERegister(t *testing.T) {
	crossMap := &crossNSEMap{}
	server := interpose.NewNetworkServiceRegistryServer(&crossMap.endpoints)

	regClient := next_reg.NewNetworkServiceEndpointRegistryClient(interpose.NewNetworkServiceEndpointRegistryClient(), adapters2.NetworkServiceEndpointServerToClient(server))
	reg, err := regClient.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "cross-nse",
		Url:  "test",
	})
	require.Nil(t, err)
	require.Greater(t, len(reg.Name), len("cross-connect-nse#"))

	_, err = server.Unregister(context.Background(), reg)
	require.Nil(t, err)
}
func TestCrossNSERegisterInvalidURL(t *testing.T) {
	crossMap := &crossNSEMap{}
	server := interpose.NewNetworkServiceRegistryServer(&crossMap.endpoints)

	regClient := next_reg.NewNetworkServiceEndpointRegistryClient(interpose.NewNetworkServiceEndpointRegistryClient(), adapters2.NetworkServiceEndpointServerToClient(server))
	req, err := regClient.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "cross-nse",
		Url:  "ht% 20", // empty URL error
	})
	require.NotNil(t, err)
	require.Nil(t, req)
}
