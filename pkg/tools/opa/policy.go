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
	"context"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	"github.com/open-policy-agent/opa/rego"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CheckAccessFunc checks rego result. Returns bool flag that means access. Returns error if something was wrong
type CheckAccessFunc func(result rego.ResultSet) (bool, error)

// CheckQueryFunc converts query string to CheckAccessFunc function
type CheckQueryFunc func(string) CheckAccessFunc

// True is default access checker, returns true if in the result set of rego exist query and it has true value
func True(query string) CheckAccessFunc {
	return func(rs rego.ResultSet) (bool, error) {
		for _, r := range rs {
			for _, e := range r.Expressions {
				if strings.HasSuffix(e.Text, query) {
					t, ok := e.Value.(bool)
					if !ok {
						return false, errors.New("policy contains non boolean expression")
					}
					return t, nil
				}
			}
		}
		return false, errors.Errorf("result is not found for query %v", query)
	}
}

// False is default access checker, returns true if in the result set of rego exist query and it has false value
func False(query string) CheckAccessFunc {
	t := True(query)
	return func(result rego.ResultSet) (bool, error) {
		b, err := t(result)
		if err == nil {
			return !b, nil
		}
		return b, err
	}
}

// WithPolicyFromSource creates custom policy based on rego source code
func WithPolicyFromSource(source, query string, checkQuery CheckQueryFunc) *AuthorizationPolicy {
	return &AuthorizationPolicy{
		policySource: strings.TrimSpace(source),
		query:        query,
		checker:      checkQuery(query),
	}
}

// WithPolicyFromFile creates custom policy based on rego source file
func WithPolicyFromFile(path, query string, checkQuery CheckQueryFunc) *AuthorizationPolicy {
	return &AuthorizationPolicy{
		policyFilePath: path,
		query:          query,
		checker:        checkQuery(query),
	}
}

// AuthorizationPolicy checks that passed tokens are valid
type AuthorizationPolicy struct {
	initErr        error
	policyFilePath string
	policySource   string
	pkg            string
	query          string
	evalQuery      *rego.PreparedEvalQuery
	checker        CheckAccessFunc
	once           sync.Once
}

// Check returns nil if passed tokens are valid
func (d *AuthorizationPolicy) Check(ctx context.Context, model interface{}) error {
	input, err := PreparedOpaInput(ctx, model)
	if err != nil {
		return err
	}
	if intErr := d.init(); intErr != nil {
		return intErr
	}
	rs, err := d.evalQuery.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	hasAccess, err := d.checker(rs)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	if !hasAccess {
		return status.Error(codes.PermissionDenied, "no sufficient privileges")
	}
	return nil
}

func (d *AuthorizationPolicy) init() error {
	d.once.Do(func() {
		if d.query == "" {
			d.query = strings.TrimSuffix(filepath.Base(d.policyFilePath), filepath.Ext(d.policyFilePath))
		}
		if d.initErr = d.loadSource(); d.initErr != nil {
			return
		}
		if d.initErr = d.checkModule(); d.initErr != nil {
			return
		}
		var r rego.PreparedEvalQuery
		r, d.initErr = rego.New(
			rego.Query(strings.Join([]string{"data", d.pkg, d.query}, ".")),
			rego.Module(d.pkg, d.policySource)).PrepareForEval(context.Background())
		if d.initErr != nil {
			return
		}
		d.evalQuery = &r
	})
	if d.initErr != nil {
		return d.initErr
	}
	if d.evalQuery == nil {
		return errors.Errorf("policy %v is not compiled", d.policySource)
	}
	return nil
}

func (d *AuthorizationPolicy) loadSource() error {
	if d.policySource != "" {
		return nil
	}
	var b []byte
	b, err := ioutil.ReadFile(d.policyFilePath)
	if err != nil {
		return err
	}
	d.policySource = strings.TrimSpace(string(b))
	return nil
}

func (d *AuthorizationPolicy) checkModule() error {
	if d.pkg != "" {
		return nil
	}
	const pkg = "package"
	lines := strings.Split(d.policySource, "\n")
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], pkg) {
			d.pkg = strings.TrimSpace(lines[i][len(pkg):])
			return nil
		}
	}
	return errors.New("missed package")
}
