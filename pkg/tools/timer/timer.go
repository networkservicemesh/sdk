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

// Package timer provides wrappers for time.AfterFunc, time.Timer with some extra information about timer and function
// state.
package timer

import "time"

// Timer is a wrapper for the time.Timer to be used with the AfterFunc.
type Timer struct {
	C     <-chan time.Time
	c     chan time.Time
	timer *time.Timer
}

// Stop prevents the Timer from firing.
// It returns true and closes the channel if the call stops the timer, false
// if the timer has already expired or been stopped.
//
// To ensure the channel is empty after a call to Stop, check the
// return value and drain the channel.
// For example, assuming the program has not received from t.C already:
//
// 	if !t.Stop() {
// 		<-t.C
// 	}
//
// This cannot be done concurrent to other receives from the Timer's
// channel or other calls to the Timer's Stop method.
func (t *Timer) Stop() (stopped bool) {
	if stopped = t.timer.Stop(); stopped {
		close(t.c)
	}
	return stopped
}

// Reset changes the timer to expire after duration d.
// It returns true if the timer had been active, false if the timer had
// expired or been stopped.
//
// Reset should be invoked only on stopped or expired timers with drained channels.
// If a program has already received a value from t.C, the timer is known
// to have expired and the channel drained, so t.Reset can be used directly.
// If a program has not yet received a value from t.C, however,
// the timer must be stopped and—if Stop reports that the timer expired
// before being stopped—the channel explicitly drained:
//
// 	if !t.Stop() {
// 		<-t.C
// 	}
// 	t.Reset(d)
//
// This should not be done concurrent to other receives from the Timer's
// channel.
//
// Note that it is not possible to use Reset's return value correctly, as there
// is a race condition between draining the channel and the new timer expiring.
// Reset should always be invoked on stopped or expired channels, as described above.
// The return value exists to preserve compatibility with existing programs.
func (t *Timer) Reset(d time.Duration) bool {
	t.c = make(chan time.Time, 1)
	t.C = t.c

	return t.timer.Reset(d)
}

// AfterFunc waits for the duration to elapse and then calls f
// in its own goroutine. It returns a Timer that can
// be used to cancel the call using its Stop method.
//
// On f completion it closes Timer.C.
func AfterFunc(d time.Duration, f func()) *Timer {
	t := &Timer{c: make(chan time.Time, 1)}
	t.C = t.c

	t.timer = time.AfterFunc(d, func() {
		f()
		close(t.c)
	})

	return t
}
