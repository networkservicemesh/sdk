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

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientinfo"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

var testData = []struct {
	name    string
	envs    map[string]string
	request *networkservice.NetworkServiceRequest
	want    *networkservice.Connection
}{
	{
		"the-labels-map-is-not-present",
		map[string]string{
			"NODE_NAME":    "AAA",
			"POD_NAME":     "BBB",
			"CLUSTER_NAME": "CCC",
		},
		&networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{},
		},
		&networkservice.Connection{
			Labels: map[string]string{
				"NodeNameKey":    "AAA",
				"PodNameKey":     "BBB",
				"ClusterNameKey": "CCC",
			},
		},
	},
	{
		"the-labels-are-overwritten",
		map[string]string{
			"NODE_NAME":    "A1",
			"POD_NAME":     "B2",
			"CLUSTER_NAME": "C3",
		},
		&networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Labels: map[string]string{
					"NodeNameKey":     "OLD_VAL1",
					"PodNameKey":      "OLD_VAL2",
					"ClusterNameKey":  "OLD_VAL3",
					"SomeOtherLabel1": "DDD",
					"SomeOtherLabel2": "EEE",
				},
			},
		},
		&networkservice.Connection{
			Labels: map[string]string{
				"NodeNameKey":     "A1",
				"PodNameKey":      "B2",
				"ClusterNameKey":  "C3",
				"SomeOtherLabel1": "DDD",
				"SomeOtherLabel2": "EEE",
			},
		},
	},
	{
		"some-of-the-envs-are-not-present",
		map[string]string{
			"CLUSTER_NAME": "ABC",
		},
		&networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Labels: map[string]string{
					"NodeNameKey":     "OLD_VAL1",
					"ClusterNameKey":  "OLD_VAL2",
					"SomeOtherLabel1": "DDD",
					"SomeOtherLabel2": "EEE",
				},
			},
		},
		&networkservice.Connection{
			Labels: map[string]string{
				"NodeNameKey":     "OLD_VAL1",
				"ClusterNameKey":  "ABC",
				"SomeOtherLabel1": "DDD",
				"SomeOtherLabel2": "EEE",
			},
		},
	},
}

func Test_clientInfo_Request(t *testing.T) {
	for _, data := range testData {
		test := data
		t.Run(test.name, func(t *testing.T) {
			testRequest(t, test.envs, test.request, test.want)
		})
	}
}

func testRequest(t *testing.T, envs map[string]string, request *networkservice.NetworkServiceRequest, want *networkservice.Connection) {
	for name, value := range envs {
		err := os.Setenv(name, value)
		if err != nil {
			t.Errorf("clientInfo.Request() unable to set up environment variable: %v", err)
		}
	}

	server := next.NewNetworkServiceClient(clientinfo.NewClient())
	got, _ := server.Request(context.Background(), request)
	assert.Equal(t, got, want)

	for name := range envs {
		err := os.Unsetenv(name)
		if err != nil {
			t.Errorf("clientInfo.Request() unable to unset environment variable: %v", err)
		}
	}
}
