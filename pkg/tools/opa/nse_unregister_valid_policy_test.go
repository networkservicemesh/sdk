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

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

func TestNSEUnregisterValidPolicy(t *testing.T) {
	var p = opa.WithNSEUnregisterValidPolicy()
	spiffeIDNSEsMap := map[string][]string{
		"id1": {"nse1", "nse2"},
		"id2": {"nse3", "nse4"},
	}

	samples := []struct {
		nseName  string
		spiffeID string
		valid    bool
	}{
		{spiffeID: "id1", nseName: "nse1", valid: true},
		{spiffeID: "id1", nseName: "nse3", valid: false},
	}

	ctx := context.Background()

	for _, sample := range samples {
		var input = authorize.RegistryOpaInput{
			SpiffeIDNSEsMap: spiffeIDNSEsMap,
			SpiffeID:        sample.spiffeID,
			ResourceName:    sample.nseName,
		}

		err := p.Check(ctx, input)
		if sample.valid {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}
