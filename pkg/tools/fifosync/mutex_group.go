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

package fifosync

import (
	"fmt"
	"sync"
)

// MutexGroup is a by-ID mapped FIFO mutex group
type MutexGroup struct {
	init  sync.Once
	lock  Mutex
	locks map[string]*mutex
}

type mutex struct {
	mutex Mutex
	count int
}

// Lock locks FIFO mutex selected by the given ID
func (g *MutexGroup) Lock(id string) {
	g.init.Do(func() {
		g.locks = map[string]*mutex{}
	})

	g.lock.Lock()

	lock, ok := g.locks[id]
	if !ok {
		lock = &mutex{}
		g.locks[id] = lock
	}
	lock.count++

	g.lock.Unlock()

	lock.mutex.Lock()
}

// Unlock unlocks FIFO mutex selected by the given ID
func (g *MutexGroup) Unlock(id string) {
	g.lock.Lock()

	lock, ok := g.locks[id]
	if !ok {
		panic(fmt.Sprintf("id is not locked: %v", id))
	}
	lock.count--
	if lock.count == 0 {
		delete(g.locks, id)
	}

	g.lock.Unlock()

	lock.mutex.Unlock()
}
