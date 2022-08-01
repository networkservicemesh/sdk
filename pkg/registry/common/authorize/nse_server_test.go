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
	"net/url"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"

	"go.uber.org/goleak"
)

func TestAuthzEndpointRegistry(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	server := authorize.NewNetworkServiceEndpointRegistryServer(
		authorize.WithRegisterPolicies(opa.WithRegisterValidPolicy()),
		authorize.WithUnregisterPolicies(opa.WithUnregisterValidPolicy()),
	)

	nseReg := &registry.NetworkServiceEndpoint{Name: "nse-1"}

	u1, _ := url.Parse("spiffe://test.com/workload1")
	u2, _ := url.Parse("spiffe://test.com/workload2")
	cert1 := generateCert(u1)
	cert2 := generateCert(u2)
	cert1Ctx, err := withPeer(context.Background(), cert1)
	require.NoError(t, err)
	cert2Ctx, err := withPeer(context.Background(), cert2)
	require.NoError(t, err)

	_, err = server.Register(cert1Ctx, nseReg)
	require.NoError(t, err)

	_, err = server.Register(cert2Ctx, nseReg)
	require.Error(t, err)

	_, err = server.Register(cert1Ctx, nseReg)
	require.NoError(t, err)

	_, err = server.Unregister(cert2Ctx, nseReg)
	require.Error(t, err)

	_, err = server.Unregister(cert1Ctx, nseReg)
	require.NoError(t, err)
}
