// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package injecterror

import (
	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

type errorSupplier struct {
	err        error
	count      atomic.Int32
	errorTimes []int
}

// supply returns an error or nil depending on errorTimes
// * [0, 2, 3] - will return an error on 0, 2, 3 times
// * [-1] - will return an error on all requests
// * [1, 4, -1] - will return an error on 0 time and on all times starting from 4.
func (e *errorSupplier) supply() error {
	defer func() { e.count.Inc() }()

	count := int(e.count.Load())
	for _, errorTime := range e.errorTimes {
		if errorTime > count {
			break
		}
		if errorTime == count || errorTime == -1 {
			return errors.WithStack(e.err)
		}
	}

	return nil
}
