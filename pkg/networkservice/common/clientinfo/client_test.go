// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientinfo"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
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

func TestClientInfo(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	tests := []struct {
		name      string
		envs      map[string]string
		oldLabels map[string]string
		want      map[string]string
	}{
		{
			name: "LabelsMapNotPresent",
			envs: map[string]string{
				"NODE_NAME":    "AAA",
				"POD_NAME":     "BBB",
				"CLUSTER_NAME": "CCC",
			},
			oldLabels: nil,
			want: map[string]string{
				"nodeName":    "AAA",
				"podName":     "BBB",
				"clusterName": "CCC",
			},
		},
		{
			name: "LabelsAlreadySet",
			envs: map[string]string{
				"NODE_NAME":    "AAA",
				"POD_NAME":     "BBB",
				"CLUSTER_NAME": "CCC",
			},
			oldLabels: map[string]string{
				"nodeName":       "OLD_VAL1",
				"podName":        "OLD_VAL2",
				"clusterName":    "OLD_VAL3",
				"SomeOtherLabel": "DDD",
			},
			want: map[string]string{
				"nodeName":       "OLD_VAL1",
				"podName":        "OLD_VAL2",
				"clusterName":    "OLD_VAL3",
				"SomeOtherLabel": "DDD",
			},
		},
		{
			name: "SomeEnvsNotPresent",
			envs: map[string]string{
				"POD_NAME":     "BBB",
				"CLUSTER_NAME": "CCC",
			},
			oldLabels: map[string]string{
				"nodeName":       "OLD_VAL1",
				"clusterName":    "OLD_VAL3",
				"SomeOtherLabel": "DDD",
			},
			want: map[string]string{
				"nodeName":       "OLD_VAL1",
				"podName":        "BBB",
				"clusterName":    "OLD_VAL3",
				"SomeOtherLabel": "DDD",
			},
		},
		{
			name: "SomeEnvsAndLabelsNotPresent",
			envs: map[string]string{
				"POD_NAME":     "BBB",
				"CLUSTER_NAME": "CCC",
			},
			oldLabels: map[string]string{
				"clusterName":    "OLD_VAL3",
				"SomeOtherLabel": "DDD",
			},
			want: map[string]string{
				"podName":        "BBB",
				"clusterName":    "OLD_VAL3",
				"SomeOtherLabel": "DDD",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := setEnvs(tc.envs)
			require.NoError(t, err)

			client := next.NewNetworkServiceClient(
				clientinfo.NewClient(),
				checkrequest.NewClient(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
					require.Equal(t, tc.want, request.GetConnection().GetLabels())
				}),
			)
			conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Labels: tc.oldLabels,
				},
			})
			require.NoError(t, err)
			require.NotNil(t, conn)

			err = unsetEnvs(tc.envs)
			require.NoError(t, err)
		})
	}
}
