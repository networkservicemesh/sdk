// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package switchcase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/switchcase"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
)

func TestSwitchServer(t *testing.T) {
	for _, s := range testSamples() {
		t.Run(s.name, func(t *testing.T) {
			//nolint:scopelint
			testSwitchServer(t, s.conditions, s.result)
		})
	}
}

func testSwitchServer(t *testing.T, conditions []switchcase.Condition, expected int) {
	var actual int

	var cases []*switchcase.ServerCase
	for i, cond := range conditions {
		cases = append(cases, &switchcase.ServerCase{
			Condition: cond,
			Server: checkcontext.NewServer(t, func(*testing.T, context.Context) {
				actual = i
			}),
		})
	}

	counter := new(count.Server)
	s := next.NewNetworkServiceServer(
		switchcase.NewServer(cases...),
		counter,
	)

	ctx := withN(context.Background(), 1)

	actual = -1
	_, err := s.Request(ctx, new(networkservice.NetworkServiceRequest))
	require.NoError(t, err)
	require.Equal(t, expected, actual)
	require.Equal(t, 1, counter.Requests())

	actual = -1
	_, err = s.Close(ctx, new(networkservice.Connection))
	require.NoError(t, err)
	require.Equal(t, expected, actual)
	require.Equal(t, 1, counter.Closes())
}
