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

package setid_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/setid"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
)

func TestGenIDNetworkServiceRegistryServer_RegisterNSE(t *testing.T) {
	chain := next.NewNetworkServiceRegistryServer(setid.NewNetworkServiceRegistryServer(), memory.NewNetworkServiceRegistryServer("nsm-1", memory.NewMemoryResourceClient()))
	nse := &registry.NetworkServiceEndpoint{
		NetworkServiceName: "ns-1",
		Payload:            "IP",
	}
	registration := &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    "ns-1",
			Payload: "IP",
		},
		NetworkServiceEndpoint: nse,
	}
	resp, err := chain.RegisterNSE(context.Background(), registration)
	require.Nil(t, err)
	require.NotEmpty(t, resp.NetworkServiceEndpoint.Name)
}
