// Copyright (c) 2022 Doc.ai and/or its affiliates.
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

package opa_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/monitor/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

func TestWithServiceConnectionPolicy(t *testing.T) {
	var p = opa.WithServiceOwnConnectionPolicy()
	var input = authorize.MonitorOpaInput{
		PathSegments: []*networkservice.PathSegment{
			{},
			{
				Id: "conn1",
			},
		},
		SpiffeIDConnectionMap: map[string][]string{
			spiffeID: {"conn1"},
		},
		ServiceSpiffeID: spiffeID,
	}

	ctx := context.Background()

	err := p.Check(ctx, input)
	require.NoError(t, err)
}
