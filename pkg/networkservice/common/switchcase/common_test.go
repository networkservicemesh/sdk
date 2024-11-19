// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2024 Cisco and/or its affiliates.
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

package switchcase_test

import (
	"context"
	"fmt"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/switchcase"
)

type sample struct {
	name       string
	conditions []switchcase.Condition
	result     int
}

func testSamples() (samples []*sample) {
	for i := 0; i < 8; i++ {
		s := &sample{
			conditions: make([]switchcase.Condition, 3),
			result:     -1,
		}

		bits := i
		for k := range s.conditions {
			bit := bits % 2
			s.conditions[k] = condition(bit)
			if bit == 1 {
				s.name = fmt.Sprint(s.name, "true ")
				if s.result == -1 {
					s.result = k
				}
			} else {
				s.name = fmt.Sprint(s.name, "false ")
			}
			bits /= 2
		}
		s.name = fmt.Sprint(s.name, s.result)

		samples = append(samples, s)
	}
	return samples
}

func condition(n int) switchcase.Condition {
	return func(ctx context.Context, _ *networkservice.Connection) bool {
		if value := ctx.Value("key"); value != nil {
			num, ok := value.(int)
			return ok && num == n
		}
		return false
	}
}

func withN(ctx context.Context, n int) context.Context {
	// nolint:staticcheck
	return context.WithValue(ctx, "key", n)
}
