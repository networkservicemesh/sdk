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
	"sync"
	"sync/atomic"
)

// NaiveMutex is a FIFO order guaranteed mutex
type NaiveMutex struct {
	init        sync.Once
	ticket      int32
	ticketInUse int32
	queueLock   sync.RWMutex
	queueCond   *sync.Cond
	mainLock    sync.Mutex
}

// Lock acquires lock or blocks until it gets free
func (m *NaiveMutex) Lock() {
	m.lock(atomic.AddInt32(&m.ticket, 1) - 1)
}

func (m *NaiveMutex) lock(ticket int32) {
	m.init.Do(func() {
		m.queueCond = sync.NewCond(m.queueLock.RLocker())
	})

	m.queueLock.RLock()
	for ticket != m.ticketInUse {
		m.queueCond.Wait()
	}
	m.queueLock.RUnlock()

	m.mainLock.Lock()

	m.queueLock.Lock()
	m.ticketInUse++
	m.queueLock.Unlock()
}

// Unlock frees lock
func (m *NaiveMutex) Unlock() {
	m.mainLock.Unlock()
	m.queueCond.Broadcast()
}
