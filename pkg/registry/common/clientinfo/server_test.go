// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
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
	"os"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clientinfo"
)

func setEnvs(envs map[string]string) error {
	for name, value := range envs {
		if err := os.Setenv(name, value); err != nil {
			return err
		}
	}
	return nil
}

func unsetEnvs(envs map[string]string) error {
	for name := range envs {
		if err := os.Unsetenv(name); err != nil {
			return err
		}
	}
	return nil
}

var testCases = []struct {
	name     string
	envs     map[string]string
	expected map[string]string
	input    map[string]string
}{
	{
		name: "MapNotPresent",
		envs: map[string]string{
			"NODE_NAME":    "AAA",
			"POD_NAME":     "BBB",
			"CLUSTER_NAME": "CCC",
		},
		expected: map[string]string{
			"nodeName":    "AAA",
			"podName":     "BBB",
			"clusterName": "CCC",
		},
	},
	{
		name: "LabelsOverwritten",
		envs: map[string]string{
			"NODE_NAME":    "AAA",
			"POD_NAME":     "BBB",
			"CLUSTER_NAME": "CCC",
		},
		expected: map[string]string{
			"nodeName":       "OLD_VAL1",
			"podName":        "OLD_VAL2",
			"clusterName":    "OLD_VAL3",
			"SomeOtherLabel": "DDD",
		},
		input: map[string]string{
			"nodeName":       "OLD_VAL1",
			"podName":        "OLD_VAL2",
			"clusterName":    "OLD_VAL3",
			"SomeOtherLabel": "DDD",
		},
	},
	{
		name: "SomeEnvsNotPresent",
		envs: map[string]string{
			"CLUSTER_NAME": "CCC",
		},
		expected: map[string]string{
			"nodeName":       "OLD_VAL1",
			"clusterName":    "OLD_VAL2",
			"SomeOtherLabel": "DDD",
		},
		input: map[string]string{
			"nodeName":       "OLD_VAL1",
			"clusterName":    "OLD_VAL2",
			"SomeOtherLabel": "DDD",
		},
	},
}

func TestLabelsServer(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testLabelsServer(t, tc.envs, tc.expected, tc.input)
		})
	}
}

func testLabelsServer(t *testing.T, envs, want, input map[string]string) {
	nsReg := registry.NetworkService{Name: "ns-1"}
	nseReg := &registry.NetworkServiceEndpoint{
		Name: "nse-1",
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{nsReg.Name: {
			Labels: input,
		}},
	}

	err := setEnvs(envs)
	require.NoError(t, err)

	server := clientinfo.NewNetworkServiceEndpointRegistryServer()

	// Register
	nseReg, err = server.Register(context.Background(), nseReg)
	require.NoError(t, err)
	require.Equal(t, want, nseReg.NetworkServiceLabels[nsReg.Name].Labels)

	// Unregister
	_, err = server.Unregister(context.Background(), nseReg)
	require.NoError(t, err)

	err = unsetEnvs(envs)
	require.NoError(t, err)
}
