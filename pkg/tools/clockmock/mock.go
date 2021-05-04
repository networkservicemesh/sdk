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

// Package clockmock provides tools for mocking time functions
package clockmock

import (
	"context"
	"sync"
	"time"

	libclock "github.com/benbjohnson/clock"

	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

var _ clock.Clock = (*Mock)(nil)

// Mock is a mock implementation of the Clock
type Mock struct {
	lock sync.RWMutex
	mock *libclock.Mock
}

// New returns a new mocked clock
func New() *Mock {
	return &Mock{
		mock: libclock.NewMock(),
	}
}

// Start starts mock time to run with the given speed until ctx becomes done. While time is running, current time for
// the mock will be the following:
//   mock time := mock start time  +  (real time duration from the start) * speed  +  mock duration added with Set, Add
func (m *Mock) Start(ctx context.Context, speed float64) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	const tick = 10 * time.Millisecond

	realStart, mockStart, mockTime := time.Now(), m.Now(), m.Now()
	var mockAdded time.Duration
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(tick):
				m.lock.Lock()

				mockAdded += m.Since(mockTime)
				mockTime = mockStart.
					Add(time.Duration(float64(time.Since(realStart)) * speed)).
					Add(mockAdded)
				m.mock.Set(mockTime)

				m.lock.Unlock()
			}
		}
	}()
}

// Set sets the current time of the mock clock to a specific one.
func (m *Mock) Set(t time.Time) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.mock.Set(t)
}

// Add moves the current time of the mock clock forward by the specified duration.
func (m *Mock) Add(d time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.mock.Add(d)
}

// Now returns mock current time
func (m *Mock) Now() time.Time {
	return m.mock.Now()
}

// Since is a shortcut for the m.Now().Sub(t)
func (m *Mock) Since(t time.Time) time.Duration {
	return m.mock.Since(t)
}

// Until is a shortcut for the t.Sub(m.Now())
func (m *Mock) Until(t time.Time) time.Duration {
	return t.Sub(m.Now())
}

// Sleep waits for the mock current time becomes > m.Now().Add(d)
func (m *Mock) Sleep(d time.Duration) {
	<-m.After(d)
}

// Timer returns a timer that will fire when the mock current time becomes > m.Now().Add(d)
func (m *Mock) Timer(d time.Duration) clock.Timer {
	if d = safeDuration(d); d > 0 {
		m.lock.RLock()
		defer m.lock.RUnlock()
	} else {
		m.lock.Lock()
		defer m.lock.Unlock()

		defer m.mock.Add(0)
	}

	return &mockTimer{
		mock:  m,
		Timer: m.mock.Timer(d),
	}
}

// After is a shortcut for the m.Timer(d).C()
func (m *Mock) After(d time.Duration) <-chan time.Time {
	return m.Timer(d).C()
}

// AfterFunc returns a timer that will call f when the mock current time becomes > m.Now().Add(d)
func (m *Mock) AfterFunc(d time.Duration, f func()) clock.Timer {
	if d = safeDuration(d); d > 0 {
		m.lock.RLock()
		defer m.lock.RUnlock()
	} else {
		m.lock.Lock()
		defer m.lock.Unlock()

		defer m.mock.Add(0)
	}

	return m.afterFunc(d, f)
}

func (m *Mock) afterFunc(d time.Duration, f func()) clock.Timer {
	return &mockTimer{
		mock: m,
		Timer: m.mock.AfterFunc(d, func() {
			go f()
		}),
	}
}

// Ticker returns a ticker that will fire every time when the mock current time becomes > mock previous time + d
func (m *Mock) Ticker(d time.Duration) clock.Ticker {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return &mockTicker{
		Ticker: m.mock.Ticker(d),
	}
}

// WithDeadline wraps parent in a new context that will cancel when the mock current time becomes > deadline
func (m *Mock) WithDeadline(parent context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	cancelCtx, cancel := context.WithCancel(parent)

	ctx := &timerCtx{
		deadline: deadline,
		Context:  cancelCtx,
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	if timeout := m.Until(deadline); timeout > 0 {
		ctx.timer = m.afterFunc(timeout, cancel)
	} else {
		cancel()
		return ctx, cancel
	}

	go func() {
		<-cancelCtx.Done()
		ctx.timer.Stop()
	}()

	return ctx, func() {
		ctx.timer.Stop()
		cancel()
	}
}

// WithTimeout is a shortcut for the m.WithDeadline(parent, m.Now().Add(timeout))
func (m *Mock) WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return m.WithDeadline(parent, m.Now().Add(timeout))
}

type timerCtx struct {
	deadline time.Time
	timer    clock.Timer

	context.Context
}

func (c *timerCtx) Deadline() (deadline time.Time, ok bool) {
	return c.deadline, true
}
