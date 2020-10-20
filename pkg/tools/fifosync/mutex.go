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
)

// Mutex is a FIFO order guaranteed mutex
type Mutex struct {
	init sync.Once
	ch   chan struct{}
}

// Lock acquires lock or blocks until it gets free
func (m *Mutex) Lock() {
	m.init.Do(func() {
		m.ch = make(chan struct{}, 1)
	})
	m.ch <- struct{}{}
}

// Unlock frees lock
func (m *Mutex) Unlock() {
	<-m.ch
}

// Mutate mutates 'locked' Mutex to 'next' Mutex:
// * if 'next' mutex is free, it first locks 'next', than unlocks 'locked'
// * if 'next' mutex is locked, it first unlocks 'locked', than locks (or starts
//   waiting on) 'next'
func Mutate(locked, next *Mutex) {
	next.init.Do(func() {
		next.ch = make(chan struct{}, 1)
	})

	select {
	case next.ch <- struct{}{}:
		<-locked.ch
	default:
		next.ch <- <-locked.ch
	}
}
