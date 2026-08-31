// Copyright (c) 2022 Cisco and/or its affiliates.
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

// Package once is an extension of sync.Once
package once

import (
	"sync"
	"sync/atomic"
)

// UntilSuccess is similar to sync.Once, but will be callable until the first success
type UntilSuccess struct {
	done uint32
	m    sync.Mutex
}

// Do will be callable until the first success
func (u *UntilSuccess) Do(f func() bool) {
	if atomic.LoadUint32(&u.done) > 0 {
		return
	}
	u.m.Lock()
	defer u.m.Unlock()
	if u.done == 0 && f() {
		atomic.StoreUint32(&u.done, 1)
	}
}
