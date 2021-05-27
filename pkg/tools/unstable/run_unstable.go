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

// Package unstable provide tool for running unstable tests with multiple attempts
package unstable

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"
)

// RunUnstable retries `f` until it becomes successful, but no more than `attempts` times. If there is successful `f`
// attempt, it wouldn't affect the result of other tests.
// Examples:
// 1. Running tests: [success_1, unstable_success, success_2]
//    Result: success
// 2. Running tests: [success_1, unstable_failure, success_2]
//    Result: failure
// 3. Running tests: [failure_1, unstable_success, success_2]
//    Result: failure
// 4. Running tests: [failure_1, unstable_failure, success_2]
//    Result: failure
// 5. Running tests: [success_1, unstable_success, failure_2]
//    Result: failure
// 6. Running tests: [success_1, unstable_failure, failure_2]
//    Result: failure
func RunUnstable(t *testing.T, attempts int, f func(t *testing.T)) bool {
	// Receive pointers to `failed` field for all parent *testing.T
	failedPtrs := getFailedPtrs(t)
	// Save parents `failed` states
	failedStates := make([]bool, len(failedPtrs))
	for j, failedPtr := range failedPtrs {
		failedStates[j] = *failedPtr
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		if t.Run(fmt.Sprintf("Unstable %d", attempt), f) {
			if attempt != 1 {
				// Restore parents `failed` states if there was more than 1 attempt
				for j, failedPtr := range failedPtrs {
					*failedPtr = failedStates[j]
				}
			}
			return true
		}
	}

	return false
}

func getFailedPtrs(t *testing.T) (failedPtrs []*bool) {
	tPtr := reflect.ValueOf(t)
	for {
		val := reflect.Indirect(tPtr)
		if !val.IsValid() || val.IsZero() {
			return failedPtrs
		}

		// nolint:gosec
		failedPtr := unsafe.Pointer(val.FieldByName("failed").UnsafeAddr())
		failedPtrs = append(failedPtrs, (*bool)(failedPtr))

		tPtr = val.FieldByName("parent")
	}
}
