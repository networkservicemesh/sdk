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

func TestLabelsMapNotPresent(t *testing.T) {
	envs := map[string]string{
		"NODE_NAME":    "AAA",
		"POD_NAME":     "BBB",
		"CLUSTER_NAME": "CCC",
	}
	expected := map[string]string{
		"NodeNameKey":    "AAA",
		"PodNameKey":     "BBB",
		"ClusterNameKey": "CCC",
	}

	err := setEnvs(envs)
	assert.Nil(t, err)

	client := clientinfo.NewClient()
	conn, err := client.Request(context.Background(), &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	})
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.Equal(t, expected, conn.GetLabels())

	err = unsetEnvs(envs)
	assert.Nil(t, err)
}

func TestLabelsOverwritten(t *testing.T) {
	envs := map[string]string{
		"NODE_NAME":    "AAA",
		"POD_NAME":     "BBB",
		"CLUSTER_NAME": "CCC",
	}
	expected := map[string]string{
		"NodeNameKey":    "AAA",
		"PodNameKey":     "BBB",
		"ClusterNameKey": "CCC",
		"SomeOtherLabel": "DDD",
	}

	err := setEnvs(envs)
	assert.Nil(t, err)

	client := clientinfo.NewClient()
	conn, err := client.Request(context.Background(), &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Labels: map[string]string{
				"NodeNameKey":    "OLD_VAL1",
				"PodNameKey":     "OLD_VAL2",
				"ClusterNameKey": "OLD_VAL3",
				"SomeOtherLabel": "DDD",
			},
		},
	})
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.Equal(t, expected, conn.GetLabels())

	err = unsetEnvs(envs)
	assert.Nil(t, err)
}

func TestSomeEnvsNotPresent(t *testing.T) {
	envs := map[string]string{
		"CLUSTER_NAME": "CCC",
	}
	expected := map[string]string{
		"NodeNameKey":    "OLD_VAL1",
		"ClusterNameKey": "CCC",
		"SomeOtherLabel": "DDD",
	}

	err := setEnvs(envs)
	assert.Nil(t, err)

	client := clientinfo.NewClient()
	conn, err := client.Request(context.Background(), &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Labels: map[string]string{
				"NodeNameKey":    "OLD_VAL1",
				"ClusterNameKey": "OLD_VAL2",
				"SomeOtherLabel": "DDD",
			},
		},
	})
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.Equal(t, expected, conn.GetLabels())

	err = unsetEnvs(envs)
	assert.Nil(t, err)
}
