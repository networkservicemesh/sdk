// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2021 Nordix Foundation.
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

package tokens_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/tokens"
)

func TestToEnv(t *testing.T) {
	name, value := tokens.ToEnv("name", []string{"1", "2", "3"})
	require.Equal(t, "NSM_SRIOV_TOKENS_name", name)
	require.Equal(t, "1,2,3", value)
}

func TestFromEnv(t *testing.T) {
	envs := []string{
		"A=aaa",
		"NSM_SRIOV_TOKENS_name-1=1,2,3",
		"B=bbb",
		"NSM_SRIOV_TOKENS_name-2=4",
	}

	toks := tokens.FromEnv(envs)
	require.Equal(t, map[string][]string{
		"name-1": {"1", "2", "3"},
		"name-2": {"4"},
	}, toks)
}
