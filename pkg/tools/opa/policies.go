// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

package opa

import (
	"embed"
	_ "embed"
	"os"

	"github.com/pkg/errors"
)

//go:embed policies/*.rego
var policiesFS embed.FS

// PolicyPath path to the policy source file
type PolicyPath string

func (p PolicyPath) Read() (*AuthorizationPolicy, error) {
	b, err := os.ReadFile(string(p))
	if err != nil {
		var embdedErr error
		b, embdedErr = policiesFS.ReadFile(string(p))
		if embdedErr != nil {
			return nil, errors.Wrap(err, embdedErr.Error())
		}
	}
	return &AuthorizationPolicy{
		policySource: string(b),
		query:        "valid",
		checker:      True("valid"),
	}, nil
}
