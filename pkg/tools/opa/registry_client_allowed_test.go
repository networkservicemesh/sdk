// Copyright (c) 2022 Cisco and/or its affiliates.
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

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

func getPath(t *testing.T, spiffeID string) *registry.Path {
	var segments = []struct {
		name           string
		tokenGenerator token.GeneratorFunc
	}{
		{
			name: spiffeID,
			tokenGenerator: genTokenFunc(&jwt.RegisteredClaims{
				Subject:  spiffeID,
				Audience: []string{"nsmgr"},
			}),
		},
		{
			name: "nsmgr",
			tokenGenerator: genTokenFunc(&jwt.RegisteredClaims{
				Subject:  "nsmgr",
				Audience: []string{"nse"},
			}),
		},
	}

	path := &registry.Path{
		PathSegments: []*registry.PathSegment{},
	}

	for _, segment := range segments {
		tok, expire, err := segment.tokenGenerator(nil)
		require.NoError(t, err)
		path.PathSegments = append(path.PathSegments, &registry.PathSegment{
			Name:    segment.name,
			Token:   tok,
			Expires: timestamppb.New(expire),
		})
	}

	return path
}
func TestRegistryClientAllowedPolicy(t *testing.T) {
	var p = opa.WithRegistryClientAllowedPolicy()
	spiffeIDResourcesMap := map[string][]string{
		"id1": {"nse1", "nse2"},
		"id2": {"nse3", "nse4"},
	}

	samples := []struct {
		nseName  string
		spiffeID string
		valid    bool
	}{
		{spiffeID: "id1", nseName: "nse3", valid: false},
		{spiffeID: "id1", nseName: "nse1", valid: true},
		{spiffeID: "id1", nseName: "nse5", valid: true},
		{spiffeID: "id3", nseName: "nse5", valid: true},
		{spiffeID: "id3", nseName: "nse2", valid: false},
	}

	ctx := context.Background()

	for _, sample := range samples {
		path := getPath(t, sample.spiffeID)
		var input = authorize.RegistryOpaInput{
			SpiffeIDResourcesMap: spiffeIDResourcesMap,
			ResourceName:         sample.nseName,
			PathSegments:         path.PathSegments,
		}

		err := p.Check(ctx, input)
		if sample.valid {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}
