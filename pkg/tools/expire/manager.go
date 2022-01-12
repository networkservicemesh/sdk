// Copyright (c) 2022 Doc.ai and/or its affiliates.
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

// Package expire provide expiration manager
package expire

import (
	"context"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

// Executor is a serialize.Executor interface
type Executor interface {
	AsyncExec(f func()) <-chan struct{}
}

// Manager manages expiration for some entities
type Manager struct {
	ctx       context.Context
	clockTime clock.Clock
	timers    timerMap
}

type timer struct {
	expirationTime time.Time
	stopped        bool

	clock.Timer
}

// NewManager creates a new Manager
func NewManager(ctx context.Context) *Manager {
	return &Manager{
		ctx:       ctx,
		clockTime: clock.FromContext(ctx),
	}
}

// New creates a new expiration for the `id`, on expiration it would call `closeFunc`
func (m *Manager) New(executor Executor, id string, expirationTime time.Time, closeFunc func(context.Context)) {
	if executor == nil {
		panic("cannot create a new expiration with nil Executor")
	}

	var t *timer
	t = &timer{
		expirationTime: expirationTime,
		Timer: m.clockTime.AfterFunc(m.clockTime.Until(expirationTime), func() {
			executor.AsyncExec(func() {
				if tt, ok := m.timers.Load(id); !ok || tt != t {
					return
				}
				m.timers.Delete(id)

				closeCtx, cancel := context.WithCancel(m.ctx)
				defer cancel()

				closeFunc(closeCtx)
			})
		}),
	}

	m.timers.Store(id, t)
}

// Stop stops expiration for the `id`
func (m *Manager) Stop(id string) bool {
	t, loaded := m.timers.Load(id)
	if loaded {
		t.stopped = t.Stop()
	}
	return loaded
}

// Start starts stopped expiration for the `id` with the same expiration time
func (m *Manager) Start(id string) {
	if t, ok := m.timers.Load(id); ok && t.stopped {
		t.stopped = false
		t.Reset(m.clockTime.Until(t.expirationTime))
	}
}

// Expire force expires stopped expiration for the `id`
func (m *Manager) Expire(id string) {
	if t, ok := m.timers.Load(id); ok && t.stopped {
		t.stopped = false
		t.Reset(0)
	}
}

// Delete deletes expiration for the `id`
func (m *Manager) Delete(id string) bool {
	t, ok := m.timers.LoadAndDelete(id)
	if !ok {
		return false
	}

	t.Stop()

	return true
}
