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
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

func TestWithAllTokensValidPolicy(t *testing.T) {
	validPath := &networkservice.Path{
		PathSegments: []*networkservice.PathSegment{
			{
				Token: genJWTWithClaimsWithYear(time.Now().Year() + 10),
			},
			{
				Token: genJWTWithClaimsWithYear(time.Now().Year() + 10),
			},
			{
				Token: genJWTWithClaimsWithYear(time.Now().Year() + 10),
			},
		},
	}

	invalidPath := &networkservice.Path{
		PathSegments: []*networkservice.PathSegment{
			{
				Token: genJWTWithClaimsWithYear(time.Now().Year() + 10),
			},
			{
				Token: "my token",
			},
			{
				Token: genJWTWithClaimsWithYear(time.Now().Year() + 10),
			},
		},
	}

	p := opa.WithTokensValidPolicy()
	err := p.Check(context.Background(), validPath)
	require.Nil(t, err)
	err = p.Check(context.Background(), invalidPath)
	require.NotNil(t, err)
}
