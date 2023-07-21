// Copyright (c) 2022 Cisco and/or its affiliates.
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

	"go.uber.org/goleak"
)

func TestNSERegistryAuthorizeClient(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() { goleak.VerifyNone(t, goleak.IgnoreCurrent()) })

	client := authorize.NewNetworkServiceEndpointRegistryClient(authorize.WithPolicies("etc/nsm/opa/registry/client_allowed.rego"))
	require.NotNil(t, client)

	nse := &registry.NetworkServiceEndpoint{Name: "nse"}
	path1 := getPath(t, spiffeid1)
	ctx1 := grpcmetadata.PathWithContext(context.Background(), path1)

	path2 := getPath(t, spiffeid2)
	ctx2 := grpcmetadata.PathWithContext(context.Background(), path2)

	nse.PathIds = []string{spiffeid1}
	_, err := client.Register(ctx1, nse)
	require.NoError(t, err)

	nse.PathIds = []string{spiffeid2}
	_, err = client.Register(ctx2, nse)
	require.Error(t, err)

	nse.PathIds = []string{spiffeid1}
	_, err = client.Register(ctx1, nse)
	require.NoError(t, err)

	nse.PathIds = []string{spiffeid2}
	_, err = client.Unregister(ctx2, nse)
	require.Error(t, err)

	nse.PathIds = []string{spiffeid1}
	_, err = client.Unregister(ctx1, nse)
	require.NoError(t, err)
}
