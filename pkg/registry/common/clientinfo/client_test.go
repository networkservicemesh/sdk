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

// Package clientinfo_test provides a tests for clientinfo
package clientinfo_test

import (
	"context"
	"os"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clientinfo"
)

var testData = []struct {
	name string
	envs map[string]string
	in   *registry.NSERegistration
	want *registry.NSERegistration
}{
	{
		"labels map is not present",
		map[string]string{
			"NODE_NAME":    "AAA",
			"POD_NAME":     "BBB",
			"CLUSTER_NAME": "CCC",
		},
		&registry.NSERegistration{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{},
		},
		&registry.NSERegistration{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Labels: map[string]string{
					"NodeNameKey":    "AAA",
					"PodNameKey":     "BBB",
					"ClusterNameKey": "CCC",
				},
			},
		},
	},
	{
		"labels are overwritten",
		map[string]string{
			"NODE_NAME":    "A1",
			"POD_NAME":     "B2",
			"CLUSTER_NAME": "C3",
		},
		&registry.NSERegistration{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Labels: map[string]string{
					"NodeNameKey":     "OLD_VAL1",
					"PodNameKey":      "OLD_VAL2",
					"ClusterNameKey":  "OLD_VAL3",
					"SomeOtherLabel1": "DDD",
					"SomeOtherLabel2": "EEE",
				},
			},
		},
		&registry.NSERegistration{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Labels: map[string]string{
					"NodeNameKey":     "A1",
					"PodNameKey":      "B2",
					"ClusterNameKey":  "C3",
					"SomeOtherLabel1": "DDD",
					"SomeOtherLabel2": "EEE",
				},
			},
		},
	},
	{
		"some of the envs are not present",
		map[string]string{
			"CLUSTER_NAME": "ABC",
		},
		&registry.NSERegistration{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Labels: map[string]string{
					"NodeNameKey":     "OLD_VAL1",
					"ClusterNameKey":  "OLD_VAL2",
					"SomeOtherLabel1": "DDD",
					"SomeOtherLabel2": "EEE",
				},
			},
		},
		&registry.NSERegistration{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Labels: map[string]string{
					"NodeNameKey":     "OLD_VAL1",
					"ClusterNameKey":  "ABC",
					"SomeOtherLabel1": "DDD",
					"SomeOtherLabel2": "EEE",
				},
			},
		},
	},
}

func Test_clientInfoRegistry_RegisterNSE(t *testing.T) {
	for _, data := range testData {
		test := data
		t.Run(test.name, func(t *testing.T) {
			testRequest(t, test.envs, test.in, test.want)
		})
	}
}

func testRequest(t *testing.T, envs map[string]string, registration, want *registry.NSERegistration) {
	for name, value := range envs {
		if err := os.Setenv(name, value); err != nil {
			t.Errorf("clientInfoRegistry.RegisterNSE() unable to set up environment variable: %v", err)
		}
	}

	client := next.NewRegistryClient(clientinfo.NewRegistryClient())
	got, _ := client.RegisterNSE(context.Background(), registration)
	assert.Equal(t, got, want)

	for name := range envs {
		if err := os.Unsetenv(name); err != nil {
			t.Errorf("clientInfoRegistry.RegisterNSE() unable to unset environment variable: %v", err)
		}
	}
}
