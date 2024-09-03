// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

const defaultPoliciesDir = "etc/nsm/opa"

//go:embed policies/**/*.rego
var embedFS embed.FS

func PoliciesByFileMask(masks ...string) ([]*AuthorizationPolicy, error) {
	var policies []*AuthorizationPolicy

	if len(masks) == 0 {
		return policies, nil
	}

	for _, mask := range masks {
		files, err := findFilesByPath(strings.ReplaceAll(mask, defaultPoliciesDir, "policies"))
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			policy, err := PolicyFromFile(file)
			if err != nil {
				return nil, err
			}
			policies = append(policies, policy)
		}
	}

	return policies, nil
}

func PolicyFromFile(p string) (*AuthorizationPolicy, error) {
	// #nosec
	b, err := os.ReadFile(p)
	if err != nil {
		var embedErr error
		b, embedErr = embedFS.ReadFile(strings.ReplaceAll(p, defaultPoliciesDir, "policies"))
		if embedErr != nil {
			return nil, errors.Wrap(err, embedErr.Error())
		}
	}
	return &AuthorizationPolicy{
		name:         p,
		policySource: string(b),
		query:        "valid",
		checker:      True("valid"),
	}, nil
}

func findFilesByPath(mask string) ([]string, error) {
	// #nosec
	if f, err := os.Open(mask); err == nil {
		if fileInfo, err := f.Stat(); err == nil && !fileInfo.IsDir() {
			return []string{mask}, nil
		}
	}

	var result []string
	set := make(map[string]struct{})
	r, err := regexp.Compile("^" + mask + "$")

	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile regexp: ^%s$", mask)
	}

	dir := filepath.Dir(mask)

	walkFS := func(dir string, fileSystem fs.FS) {
		_ = fs.WalkDir(fileSystem, ".", func(p string, d fs.DirEntry, err error) error {
			if d == nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}

			p = path.Join(dir, p)

			if _, ok := set[p]; !ok && r.MatchString(p) {
				result = append(result, p)
				set[p] = struct{}{}
				return nil
			}

			return nil
		})
	}

	walkFS(dir, os.DirFS(dir))
	walkFS(".", embedFS)

	return result, nil
}
