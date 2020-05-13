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

package memory_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

func TestMemoryNetworkServiceDiscoveryServer_FindNetworkService(t *testing.T) {
	m := memory.NewMemoryResourceClient()
	nsm := &registry.NetworkServiceManager{
		Name: "nsm-1",
	}
	m.NetworkServiceManagers().Put(nsm)
	nse := &registry.NetworkServiceEndpoint{
		Name:                      "nse-1",
		NetworkServiceName:        "ns-1",
		NetworkServiceManagerName: "nsm-1",
	}
	m.NetworkServiceEndpoints().Put(nse)
	ns := &registry.NetworkService{
		Name: "ns-1",
	}
	m.NetworkServices().Put(ns)
	s := next.NewDiscoveryServer(memory.NewNetworkServiceDiscoveryServer(m))
	response, err := s.FindNetworkService(context.Background(), &registry.FindNetworkServiceRequest{NetworkServiceName: "ns-2"})
	require.Nil(t, response)
	require.NotNil(t, err)
	response, err = s.FindNetworkService(context.Background(), &registry.FindNetworkServiceRequest{NetworkServiceName: "ns-1"})
	require.NotNil(t, response)
	require.Nil(t, err)
	require.EqualValues(t, map[string]*registry.NetworkServiceManager{
		nsm.Name: nsm,
	}, response.NetworkServiceManagers)
	require.Equal(t, ns, response.NetworkService)
	require.EqualValues(t, []*registry.NetworkServiceEndpoint{nse}, response.NetworkServiceEndpoints)
}
