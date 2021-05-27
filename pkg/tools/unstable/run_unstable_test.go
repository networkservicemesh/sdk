// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package unstable_test

import (
	"testing"

	"github.com/networkservicemesh/sdk/pkg/tools/unstable"
)

func createTest(failureTimes ...int) func(t *testing.T) {
	var runTime int
	return func(t *testing.T) {
		defer func() { runTime++ }()

		for _, time := range failureTimes {
			if time > runTime {
				return
			}
			if time == runTime || time == -1 {
				t.FailNow()
			}
		}
	}
}

func TestRunUnstable_Stable(t *testing.T) {
	unstable.RunUnstable(t, 5, createTest())
}

func TestRunUnstable_Unstable(t *testing.T) {
	unstable.RunUnstable(t, 4, createTest(0, 1, 2))
}
