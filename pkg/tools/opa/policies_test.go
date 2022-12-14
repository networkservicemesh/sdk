// Copyright (c) 2022 Cisco Systems, Inc.
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
	"testing"

	"github.com/networkservicemesh/sdk/pkg/tools/opa"
	"github.com/stretchr/testify/require"
)

func TestDefaultPoliciesFromMask(t *testing.T) {
	policies, err := opa.PoliciesFromMasks("policies/.*.rego")
	require.NoError(t, err)
	// TODO: Test fails, because we read default policies two times in readAllFilesFromMask function. Probably, it will work in cmd-repos
	require.Len(t, policies, 7)
}

func TestCustomPolicies(t *testing.T) {
	policies, err := opa.PoliciesFromMasks("mock_policies/.*.rego")
	require.NoError(t, err)
	require.Len(t, policies, 2)
}

func TestMaskForPolicyFiles(t *testing.T) {
	policies, err := opa.PoliciesFromMasks("policies/tokens_.*.rego")
	// TODO: This test also fails, because we read matched policies two times in readAllFilesFromMask function
	require.NoError(t, err)
	require.Len(t, policies, 3)
}

func TestReadOnePolicy(t *testing.T) {
	policies, err := opa.PoliciesFromMasks("mock_policies/custom_policy.rego")
	require.NoError(t, err)
	require.Len(t, policies, 1)
}

func TestDefaultAndCustomPolicies(t *testing.T) {
	policies, err := opa.PoliciesFromMasks("policies/.*.rego", "mock_policies/.*.rego")
	require.NoError(t, err)
	require.Len(t, policies, 9)
}

// TODO: How is this supposed to work?
func TestOverriddenPolicy(t *testing.T) {
	policies, err := opa.PoliciesFromMasks("policies/.*.rego", "mock_policies/next_token_signed.rego")
	require.NoError(t, err)
	require.Len(t, policies, 7)
}
