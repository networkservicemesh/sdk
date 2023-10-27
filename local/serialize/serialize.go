// Copyright (c) 2020 Cisco Systems, Inc.
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

// Package serialize provides a simple means for Async or Sync execution of a func()
// with the guarantee that each func() will be executed exactly once and that all funcs()
// will be executed in order
package serialize

import (
	"context"
	"sync"
	"sync/atomic"
)

type job struct {
	f      func()
	done   chan struct{}
	ticket uintptr
}

// Executor - a struct that can be used to guarantee exclusive, in order execution of functions.
type Executor struct {
	buffer []*job
	mu     sync.Mutex
	ticket uintptr
}

// AsyncExec - guarantees f() will be executed Exclusively and in the Order submitted.
//
//	It immediately returns a channel that will be closed when f() has completed execution.
func (e *Executor) AsyncExec(f func()) <-chan struct{} {
	// Get a ticket.  The ticket established absolute order.
	ticket := atomic.AddUintptr(&e.ticket, 1)
	if ticket == 0 {
		panic("ticket == 0 - you've overrun a uintptr counter of jobs and are probably deadlocked")
	}
	// Create the job
	jb := &job{
		f:      f,
		done:   make(chan struct{}),
		ticket: ticket,
	}
	// The first ticket fires off processing
	if ticket == 1 {
		go e.process(context.Background(), "", jb)
		return jb.done
	}
	// queue up the job in the buffer (note: buffer order itself does not guarantee order, job.ticket does)
	e.mu.Lock()
	e.buffer = append(e.buffer, jb)
	e.mu.Unlock()
	return jb.done
}

func (e *Executor) AsyncExecContext(ctx context.Context, threadID string, f func()) <-chan struct{} {
	// Get a ticket.  The ticket established absolute order.
	ticket := atomic.AddUintptr(&e.ticket, 1)
	if ticket == 0 {
		panic("ticket == 0 - you've overrun a uintptr counter of jobs and are probably deadlocked")
	}
	// Create the job
	jb := &job{
		f:      f,
		done:   make(chan struct{}),
		ticket: ticket,
	}

	// The first ticket fires off processing
	if ticket == 1 {
		go e.process(ctx, threadID, jb)
		return jb.done
	}

	// queue up the job in the buffer (note: buffer order itself does not guarantee order, job.ticket does)
	e.mu.Lock()
	e.buffer = append(e.buffer, jb)
	e.mu.Unlock()
	return jb.done
}

func (e *Executor) process(ctx context.Context, threadID string, jb *job) {
	// Run the first job inline with processing.  This is a performance optimization
	jb.f()
	close(jb.done)
	// If there are no more jobs, exit
	if atomic.CompareAndSwapUintptr(&e.ticket, 1, 0) {
		return
	}

	// Starting from ticket == 2 (because we processed ticket == 1 already)
	ticket := uintptr(2)
	var buf []*job
	for {
		// Drain the buffer
		e.mu.Lock()
		buf = append(buf, e.buffer[0:]...)
		e.buffer = e.buffer[len(e.buffer):]
		e.mu.Unlock()
		// Sort the buffer
		for i := 0; i < len(buf); {
			if buf[i].ticket == ticket {
				buf[i].f()
				close(buf[i].done)
				if atomic.CompareAndSwapUintptr(&e.ticket, ticket, 0) {
					return
				}
				buf[i] = buf[0]
				buf = buf[1:]
				ticket++
				i = 0
				continue
			}
			i++
		}
	}
}
