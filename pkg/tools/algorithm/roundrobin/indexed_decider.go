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

// Package roundrobin provides a selection by round robins algorithm
package roundrobin

import (
	"errors"
	"sync/atomic"
)

// IndexedDecider contains index
type IndexedDecider struct {
	index int32
}

// Decide selected item from options
func (d *IndexedDecider) Decide(options ...interface{}) (interface{}, error) {
	if len(options) == 0 {
		return "", errors.New("no options provided")
	}
	atomic.AddInt32(&d.index, 1)
	return options[atomic.LoadInt32(&d.index)%int32(len(options))], nil
}
