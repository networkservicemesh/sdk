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

// Package fifosync provides FIFO order guaranteed synchronization primitives
package fifosync

import (
	"sync"
	"sync/atomic"
)

const (
	closestCount = 30
)

// Mutex is a FIFO order guaranteed mutex
type Mutex struct {
	init         sync.Once
	ticket       int32
	ticketInUse  int32
	closest      int32
	closestCount int32
	queueLock    sync.RWMutex
	queueCond    *sync.Cond
	closestLock  sync.RWMutex
	closestCond  *sync.Cond
	mainLock     sync.Mutex
}

// Lock acquires lock or blocks until it gets free
func (m *Mutex) Lock() {
	m.lock(atomic.AddInt32(&m.ticket, 1) - 1)
}

func (m *Mutex) lock(ticket int32) {
	m.init.Do(func() {
		m.closest = closestCount
		m.closestCount = closestCount
		m.queueCond = sync.NewCond(m.queueLock.RLocker())
		m.closestCond = sync.NewCond(m.closestLock.RLocker())
	})

	// wait queue for all waiters exclude the closest
	m.queueLock.RLock()
	for ticket > m.closest {
		m.queueCond.Wait()
	}
	m.queueLock.RUnlock()

	// wait queue for the closest waiters
	m.closestLock.RLock()
	for ticket != m.ticketInUse {
		m.closestCond.Wait()
	}
	m.closestLock.RUnlock()

	m.mainLock.Lock()

	// update actual ticket in use
	m.closestLock.Lock()
	m.ticketInUse++
	m.closestLock.Unlock()

	// update closest count
	m.closestCount--
	if m.closestCount == 0 {
		m.closestCount = closestCount

		// update closest
		m.queueLock.Lock()
		m.closest += closestCount
		m.queueLock.Unlock()

		m.queueCond.Broadcast()
	}
}

// Unlock frees lock
func (m *Mutex) Unlock() {
	m.mainLock.Unlock()
	m.closestCond.Broadcast()
}

// Mutate mutates 'locked' Mutex to 'next' Mutex
func Mutate(locked, next *Mutex) {
	ticket := atomic.AddInt32(&next.ticket, 1) - 1

	locked.Unlock()

	next.lock(ticket)
}
