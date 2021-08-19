// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package authorize

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

// Policy represents authorization policy for network service.
type Policy interface {
	// Check checks authorization
	Check(ctx context.Context, input interface{}) error
}

type policiesList []Policy

func (l *policiesList) check(ctx context.Context, p *networkservice.Path) error {
	if l == nil {
		return nil
	}
	for _, policy := range *l {
		if policy == nil {
			continue
		}
		if err := policy.Check(ctx, p); err != nil {
			return err
		}
	}
	return nil
}
