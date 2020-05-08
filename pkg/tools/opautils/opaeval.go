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


package opautils

import (
	"context"
	"errors"
	"github.com/open-policy-agent/opa/rego"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CheckPolicy(ctx context.Context, p *rego.PreparedEvalQuery, input interface{}) error {
	rs, err := p.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	hasAccess, err := hasAccess(rs)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if !hasAccess {
		return status.Error(codes.PermissionDenied, "no sufficient privileges to call Request")
	}

	return nil
}

func hasAccess(rs rego.ResultSet) (bool, error) {
	for _, r := range rs {
		for _, e := range r.Expressions {
			t, ok := e.Value.(bool)
			if !ok {
				return false, errors.New("policy contains non boolean expression")
			}

			if !t {
				return false, nil
			}
		}
	}

	return true, nil
}
