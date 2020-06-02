// Copyright (c) 2020 Doc.ai and/or its affiliates.
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
	"io/ioutil"
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

		default allow = true
	`

	policyPath := filepath.Clean(path.Join(dir, "policy.rego"))
	err = ioutil.WriteFile(policyPath, []byte(policy), os.ModePerm)
	require.Nil(t, err)

	p := opa.WithPolicyFromFile(policyPath, "allow", opa.True)

	err = p.Check(context.Background(), nil)
	require.Nil(t, err)
}
