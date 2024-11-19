// Copyright (c) 2022-2024 Cisco and/or its affiliates.
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

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

type testInput struct {
	ResourceID         string                      `json:"resource_id"`
	ResourceName       string                      `json:"resource_name"`
	ResourcePathIDsMap map[string][]string         `json:"resource_path_ids_map"`
	PathSegments       []*grpcmetadata.PathSegment `json:"path_segments"`
	Index              uint32                      `json:"index"`
}

func TestRegistryClientAllowedPolicy(t *testing.T) {
	p, err := opa.PolicyFromFile("etc/nsm/opa/registry/client_allowed.rego")
	require.NoError(t, err)

	resourcePathIDsMap := map[string][]string{
		"nse1": {"id1", "id2"},
	}

	samples := []struct {
		name  string
		id    string
		valid bool
	}{
		{id: "id1", name: "nse1", valid: true},
		{id: "id3", name: "nse1", valid: false},
		{id: "id1", name: "nse3", valid: true},
	}

	ctx := context.Background()

	for _, sample := range samples {
		var input = testInput{
			ResourcePathIDsMap: resourcePathIDsMap,
			ResourceName:       sample.name,
			ResourceID:         sample.id,
		}

		err := p.Check(ctx, input)
		if sample.valid {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}
