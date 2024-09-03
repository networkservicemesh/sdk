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

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

func TestDefaultPoliciesFromMask(t *testing.T) {
	policies, err := opa.PoliciesByFileMask("etc/nsm/opa/.*.rego")
	require.NoError(t, err)
	require.Len(t, policies, 7)
}

func TestCustomPolicies(t *testing.T) {
	policies, err := opa.PoliciesByFileMask("sample_policies/.*.rego")
	require.NoError(t, err)
	require.Len(t, policies, 2)
}

func TestMaskForPolicyFiles(t *testing.T) {
	policies, err := opa.PoliciesByFileMask("etc/nsm/opa/common/tokens_.*.rego")
	require.NoError(t, err)
	require.Len(t, policies, 3)
}

func TestReadOnePolicy(t *testing.T) {
	policies, err := opa.PoliciesByFileMask("sample_policies/custom_policy.rego")
	require.NoError(t, err)
	require.Len(t, policies, 1)
}

func TestDefaultAndCustomPolicies(t *testing.T) {
	policies, err := opa.PoliciesByFileMask("policies/.*.rego", "sample_policies/.*.rego")
	require.NoError(t, err)
	require.Len(t, policies, 9)
}

func TestOverriddenPolicy(t *testing.T) {
	policies, err := opa.PoliciesByFileMask("policies/.*.rego", "sample_policies/next_token_signed.rego")
	require.NoError(t, err)
	require.Len(t, policies, 8)
}

func Test_NoPoliciesPassed(t *testing.T) {
	policies, err := opa.PoliciesByFileMask()
	require.NoError(t, err)
	require.Len(t, policies, 0)
}
