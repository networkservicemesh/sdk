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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
)

func TestNsmMemoryNetworkServerRegistry_RegisterNSM(t *testing.T) {
	m := &memory.Storage{}
	s := next.NewNSMRegistryServer(memory.NewNSMRegistryServer(m, "nsm-1"))
	nsm, err := s.RegisterNSM(context.Background(), &registry.NetworkServiceManager{})
	require.Nil(t, err)
	require.NotNil(t, nsm)
	actual, _ := m.NetworkServiceManagers.Load("nsm-1")
	require.Equal(t, nsm, actual)
}

func TestNsmMemoryNetworkServerRegistry_GetEndpoints(t *testing.T) {
	m := &memory.Storage{}
	nsm := &registry.NetworkServiceManager{
		Name: "nsm-1",
	}
	m.NetworkServiceManagers.Store(nsm.Name, nsm)
	endpoints := []*registry.NetworkServiceEndpoint{
		{
			Name:                      "nse-1",
			NetworkServiceName:        "ns-1",
			NetworkServiceManagerName: "nsm-1",
		},
		{
			Name:                      "nse-2",
			NetworkServiceName:        "ns-2",
			NetworkServiceManagerName: "nsm-1",
		},
	}
	for _, e := range endpoints {
		m.NetworkServiceEndpoints.Store(e.Name, e)
	}

	s := next.NewNSMRegistryServer(memory.NewNSMRegistryServer(m, "nsm-1"))
	list, err := s.GetEndpoints(context.Background(), new(empty.Empty))
	require.Nil(t, err)
	require.NotNil(t, list)
	require.ElementsMatch(t, endpoints, list.NetworkServiceEndpoints)
}
