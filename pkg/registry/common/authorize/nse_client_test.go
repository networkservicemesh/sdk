// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package authorize_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/count"

	"go.uber.org/goleak"
)

func TestNSERegistryAuthorizeClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	callCounter := &count.CallCounter{}
	client := chain.NewNetworkServiceEndpointRegistryClient(
		authorize.NewNetworkServiceEndpointRegistryClient(authorize.WithPolicies("etc/nsm/opa/registry/client_allowed.rego")),
		count.NewNetworkServiceEndpointRegistryClient(callCounter),
	)

	nse := &registry.NetworkServiceEndpoint{Name: "nse"}
	path1 := getPath(t, spiffeid1)
	ctx1 := grpcmetadata.PathWithContext(context.Background(), path1)

	path2 := getPath(t, spiffeid2)
	ctx2 := grpcmetadata.PathWithContext(context.Background(), path2)

	nse.PathIds = []string{spiffeid1}
	_, err := client.Register(ctx1, nse)
	require.NoError(t, err)
	require.Equal(t, callCounter.Registers(), 1)

	nse.PathIds = []string{spiffeid2}
	_, err = client.Register(ctx2, nse)
	require.Error(t, err)
	require.Equal(t, callCounter.Registers(), 2)
	require.Equal(t, callCounter.Unregisters(), 0)

	nse.PathIds = []string{spiffeid1}
	_, err = client.Register(ctx1, nse)
	require.NoError(t, err)
	require.Equal(t, callCounter.Registers(), 3)
	require.Equal(t, callCounter.Unregisters(), 0)

	nse.PathIds = []string{spiffeid2}
	_, err = client.Unregister(ctx2, nse)
	require.Error(t, err)
	require.Equal(t, callCounter.Unregisters(), 1)

	nse.PathIds = []string{spiffeid1}
	_, err = client.Unregister(ctx1, nse)
	require.NoError(t, err)
	require.Equal(t, callCounter.Unregisters(), 2)
}
