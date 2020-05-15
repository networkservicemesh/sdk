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

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
)

func TestMemoryNetworkServeRegistry_RegisterNSE(t *testing.T) {
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
	ns := &registry.NetworkService{
		Name: "ns-1",
	}
	m.NetworkServices().Put(ns)
	s := next.NewNetworkServiceRegistryServer(memory.NewNetworkServiceRegistryServer("nsm-1", m))
	resp, err := s.RegisterNSE(context.Background(), nil)
	require.Nil(t, resp)
	require.NotNil(t, err)
	resp, err = s.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkService:         ns,
		NetworkServiceEndpoint: nse,
	})
	require.Nil(t, err)
	require.Equal(t, nsm, resp.NetworkServiceManager)
	require.Equal(t, nsm.Name, resp.NetworkServiceEndpoint.NetworkServiceManagerName)
	require.NotNil(t, m.NetworkServiceEndpoints().Get(nse.Name))
	require.NotNil(t, m.NetworkServices().Get(ns.Name))
}

func TestMemoryNetworkServeRegistry_RemoveNSE(t *testing.T) {
	m := memory.NewMemoryResourceClient()
	nse := &registry.NetworkServiceEndpoint{
		Name:                      "nse-1",
		NetworkServiceName:        "ns-1",
		NetworkServiceManagerName: "nsm-1",
	}
	m.NetworkServiceEndpoints().Put(nse)
	s := next.NewNetworkServiceRegistryServer(memory.NewNetworkServiceRegistryServer("nsm-1", m))
	_, err := s.RemoveNSE(context.Background(), &registry.RemoveNSERequest{NetworkServiceEndpointName: "nse-1"})
	require.Nil(t, err)
	require.Empty(t, m.NetworkServiceEndpoints().GetAll())
}
