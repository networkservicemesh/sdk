// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// Policy represents authorization policy for monitor connection.
type Policy interface {
	// Name returns policy name
	Name() string
	// Check checks authorization
	Check(ctx context.Context, input interface{}) error
}

type policiesList []Policy

func (l *policiesList) check(ctx context.Context, srv MonitorOpaInput) error {
	if l == nil {
		return nil
	}
	for _, policy := range *l {
		if policy == nil {
			continue
		}
		if err := policy.Check(ctx, srv); err != nil {
			log.FromContext(ctx).Errorf("policy failed: %v", policy.Name())
			return errors.Wrapf(err, "monitor: an error occurred during authorization policy check")
		}
	}
	return nil
}
