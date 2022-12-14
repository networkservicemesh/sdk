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
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

//go:embed policies/*.rego
var policiesFS embed.FS

// PolicyPath path to the policy source file
type PolicyPath string

func Read(paths ...string) ([]*AuthorizationPolicy, error) {
	policies := make([]*AuthorizationPolicy, 0)

	for _, path := range paths {
		embeddedPolicy, err := readEmbeddedPolicyFile(path)
		if err == nil {
			policies = append(policies, embeddedPolicy)
			continue
		}

		fileInfo, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		if fileInfo.IsDir() {
			dirPolicies, err := readPolicyDir(path)
			if err != nil {
				return nil, err
			}
			policies = append(policies, dirPolicies...)
		} else {
			policy, err := readPolicyFile(path)
			if err != nil {
				return nil, err
			}
			policies = append(policies, policy)
		}
	}

	return policies, nil
}

func readPolicyDir(path string) ([]*AuthorizationPolicy, error) {
	policies := make([]*AuthorizationPolicy, 0)

	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		policy, err := readPolicyFile(filepath.Join(path, file.Name()))
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}

	return policies, nil
}

func readEmbeddedPolicyFile(path string) (*AuthorizationPolicy, error) {
	b, err := policiesFS.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, err.Error())
	}

	return &AuthorizationPolicy{
		policySource: string(b),
		query:        "valid",
		checker:      True("valid"),
	}, nil
}

func readPolicyFile(path string) (*AuthorizationPolicy, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return &AuthorizationPolicy{
		policySource: string(b),
		query:        "valid",
		checker:      True("valid"),
	}, nil
}
