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
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
)

//go:embed policies/*.rego
var policiesFS embed.FS

func PoliciesFromMasks(masks ...string) ([]*AuthorizationPolicy, error) {
	var policies []*AuthorizationPolicy

	for _, mask := range masks {
		files, err := readAllFilesFromMask(mask)
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

func PolicyFromFile(path string) (*AuthorizationPolicy, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		var embedErr error
		b, embedErr = policiesFS.ReadFile(path)
		if embedErr != nil {
			return nil, errors.Wrap(err, embedErr.Error())
		}
	}
	return &AuthorizationPolicy{
		policySource: string(b),
		query:        "valid",
		checker:      True("valid"),
	}, nil
}

func readAllFilesFromMask(mask string) ([]string, error) {
	if fd, err := os.Open(mask); err != nil {
		fileInfo, err := fd.Stat()

		if err == nil {
			if !fileInfo.IsDir() {
				return []string{mask}, nil
			}
		}
	}

	var result []string
	r, err := regexp.Compile("^" + mask + "$")
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(mask)

	localFS := os.DirFS(dir)
	fs.WalkDir(localFS, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		completePath := filepath.Join(dir, path)
		if r.MatchString(completePath) {
			result = append(result, completePath)
			return nil
		}

		return nil
	})

	fs.WalkDir(policiesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		if r.MatchString(path) {
			result = append(result, path)
		}
		return nil
	})

	return result, nil
}
