// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
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
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

func TestWithPolicyFromFile(t *testing.T) {
	dir := filepath.Clean(path.Join(os.TempDir(), t.Name()))
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	err := os.MkdirAll(dir, os.ModePerm)
	require.Nil(t, err)

	const policy = `
package test

default valid = true
`

	policyPath := filepath.Clean(path.Join(dir, "policy.rego"))
	err = os.WriteFile(policyPath, []byte(policy), os.ModePerm)
	require.Nil(t, err)

	p, err := opa.PolicyFromFile(policyPath)
	require.NoError(t, err)

	err = p.Check(context.Background(), nil)
	require.Nil(t, err)
}
