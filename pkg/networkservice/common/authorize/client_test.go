// Copyright (c) 2023 Cisco and/or its affiliates.
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
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func TestAuthClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dir := filepath.Clean(path.Join(os.TempDir(), t.Name()))
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	err := os.MkdirAll(dir, os.ModePerm)
	require.Nil(t, err)

	policyPath := filepath.Clean(path.Join(dir, "policy.rego"))
	err = os.WriteFile(policyPath, []byte(testPolicy()), os.ModePerm)
	require.Nil(t, err)

	suits := []struct {
		name       string
		policyPath string
		request    *networkservice.NetworkServiceRequest
		response   *networkservice.Connection
		denied     bool
	}{
		{
			name:       "simple positive test",
			policyPath: policyPath,
			request:    requestWithToken("allowed"),
			denied:     false,
		},
		{
			name:       "simple negative test",
			policyPath: policyPath,
			request:    requestWithToken("not_allowed"),
			denied:     true,
		},
	}

	for i := range suits {
		s := suits[i]
		t.Run(s.name, func(t *testing.T) {
			client := chain.NewNetworkServiceClient(
				metadata.NewClient(),
				authorize.NewClient(authorize.WithPolicies(s.policyPath)),
			)
			checkResult := func(err error) {
				if !s.denied {
					require.Nil(t, err, "request expected to be not denied: ")
					return
				}
				require.NotNil(t, err, "request expected to be denied")
				s, ok := status.FromError(errors.Cause(err))
				require.True(t, ok, "error without error status code"+err.Error())
				require.Equal(t, s.Code(), codes.PermissionDenied, "wrong error status code")
			}

			_, err := client.Request(context.Background(), s.request)
			checkResult(err)

			_, err = client.Close(context.Background(), s.request.GetConnection())
			checkResult(err)
		})
	}
}

func TestAuthClientFailedRefresh(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dir := filepath.Clean(path.Join(os.TempDir(), t.Name()))
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	err := os.MkdirAll(dir, os.ModePerm)
	require.Nil(t, err)

	policyPath := filepath.Clean(path.Join(dir, "policy.rego"))
	err = os.WriteFile(policyPath, []byte(testPolicy()), os.ModePerm)
	require.Nil(t, err)

	counter := new(count.Client)
	client := chain.NewNetworkServiceClient(
		metadata.NewClient(),
		authorize.NewClient(authorize.WithPolicies(policyPath)),
		counter,
	)

	conn, err := client.Request(context.Background(), requestWithToken("allowed"))
	require.Nil(t, err)

	refreshRequest := requestWithToken("not_allowed")
	_, err = client.Request(context.Background(), refreshRequest)
	require.NotNil(t, err)
	require.Equal(t, 0, counter.Closes())

	_, err = client.Close(context.Background(), conn)
	require.Nil(t, err)
	require.Equal(t, 1, counter.Closes())
}
