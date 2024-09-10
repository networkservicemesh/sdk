// Copyright (c) 2022-2024 Cisco and/or its affiliates.
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

// Package clientinfo_test provides a tests for clientinfo
package clientinfo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clientinfo"
)

func TestLabelsClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testLabelsClient(t, tc.envs, tc.expected, tc.input)
		})
	}
}

func testLabelsClient(t *testing.T, envs, want, input map[string]string) {
	nsReg := registry.NetworkService{Name: "ns-1"}
	nseReg := &registry.NetworkServiceEndpoint{
		Name: "nse-1",
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{nsReg.Name: {
			Labels: input,
		}},
	}

	err := setEnvs(envs)
	require.NoError(t, err)

	client := clientinfo.NewNetworkServiceEndpointRegistryClient()

	// Register
	nseReg, err = client.Register(context.Background(), nseReg)
	require.NoError(t, err)
	require.Equal(t, want, nseReg.NetworkServiceLabels[nsReg.Name].Labels)

	// Unregister
	_, err = client.Unregister(context.Background(), nseReg)
	require.NoError(t, err)

	err = unsetEnvs(envs)
	require.NoError(t, err)
}
