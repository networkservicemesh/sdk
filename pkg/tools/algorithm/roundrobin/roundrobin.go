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
	"sync/atomic"
)

// RoundRobin contains index
type RoundRobin struct {
	index uint32
}

// Index returns a new index for the given size
func (r *RoundRobin) Index(size int) int {
	return int((atomic.AddUint32(&r.index, 1) - 1) % uint32(size))
}
