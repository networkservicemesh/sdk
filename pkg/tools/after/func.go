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

// Package after provides wrapper for the `time.AfterFunc` with some extensions.
package after

import (
	"context"
	"time"

	"github.com/edwarnicke/serialize"
)

// Func should execute some given function on some given time.
type Func struct {
	ctx            context.Context
	expirationTime time.Time
	fun            func()
	executor       serialize.Executor
	timer          *timer
}

type timer struct {
	started  bool
	canceled bool
	stopped  bool

	*time.Timer
}

// NewFunc creates a new Func that will execute `fun` on `expirationTime` if `ctx` won't be closed at that time.
func NewFunc(ctx context.Context, expirationTime time.Time, fun func()) *Func {
	f := &Func{
		ctx:            ctx,
		expirationTime: expirationTime,
		fun:            fun,
	}

	f.newTimer()

	return f
}

func (f *Func) newTimer() {
	timer := new(timer)
	timer.Timer = time.AfterFunc(time.Until(f.expirationTime), func() {
		f.executor.AsyncExec(func() {
			timer.started = true
			if timer.canceled || f.ctx.Err() != nil {
				return
			}
			f.fun()
		})
	})

	f.timer = timer
}

// Stop is trying to prevent Func execution:
//    * it stops Func timer and returns `true` if Func didn't yet start
//    * it waits for the Func to finish and returns `false` if Func is already executing
//    * it returns `false` if Func has been already finished
func (f *Func) Stop() bool {
	if f.timer.stopped || f.timer.canceled {
		return true
	}

	if f.timer.stopped = f.timer.Stop(); f.timer.stopped {
		return true
	}

	var stopped bool
	<-f.executor.AsyncExec(func() {
		stopped = !f.timer.started
		if stopped {
			f.timer.canceled = true
		}
	})
	return stopped
}

// Resume resumes Func timer stopped by the Stop. If Func timer hasn't been stopped, it does nothing.
func (f *Func) Resume() {
	switch {
	case f.timer.stopped:
		f.timer.stopped = false
		// Timer has been stopped, we need only to reset it.
		f.timer.Reset(time.Until(f.expirationTime))
	case f.timer.canceled:
		// Timer function has been stopped with the `canceled` flag, we need to remove the flag.
		<-f.executor.AsyncExec(func() {
			f.timer.canceled = false
			if f.timer.started {
				// Timer function has been already finished, we need to reset the timer right now.
				f.timer.Reset(0)
			}
		})
	}
}

// Reset stops Func with Stop and schedules its start on `expirationTime`.
// NOTE: if Func has already been executed, it will start it again. Use `if f.Stop() { f.Reset() }` to prevent such
//       case if it is needed.
func (f *Func) Reset(expirationTime time.Time) {
	stopped := f.Stop()

	f.expirationTime = expirationTime

	switch {
	case !stopped, f.timer.stopped:
		f.timer.stopped = false
		f.timer.Reset(time.Until(expirationTime))
	case f.timer.canceled:
		f.newTimer()
	}
}
