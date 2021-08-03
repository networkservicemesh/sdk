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

package opa_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

func TestWithPolicyFromFile(t *testing.T) {
	p := opa.WithPolicyFromFile("allow.rego", "allow", opa.True)

	err := p.Check(context.Background(), nil)
	require.NoError(t, err)
}

func TestAuthorizationPolicy_UnmarshalText(t *testing.T) {
	p1, p2 := new(opa.AuthorizationPolicy), new(opa.AuthorizationPolicy)

	err := p1.UnmarshalText([]byte(`check_name.rego:check_name:false`))
	require.NoError(t, err)

	err = p2.UnmarshalText([]byte(`allow.rego:allow:true`))
	require.NoError(t, err)

	input := &networkservice.Path{
		PathSegments: []*networkservice.PathSegment{
			{
				Name: "a",
			},
			{
				Name: "b",
			},
		},
	}
	require.NoError(t, p1.Check(context.Background(), input))
	require.NoError(t, p2.Check(context.Background(), input))

	input.PathSegments = append(input.PathSegments, &networkservice.PathSegment{
		Name: "test-name",
	})
	require.Error(t, p1.Check(context.Background(), input))
	require.NoError(t, p2.Check(context.Background(), input))
}
