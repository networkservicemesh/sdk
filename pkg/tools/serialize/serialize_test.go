// Copyright (c) 2020 Cisco and/or its affiliates.
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

// Package serialize_test tests the contracts of the serialize.Executor which are:
//  1.  One at a time - the executor will never execute more than one func()
//                         provided to it at a time
//  2.  In Order - the order of execution of func()s provided to it will always be preserved: first in, first executed
package serialize_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
)

func TestNestedAsyncExec(t *testing.T) {
	var exec serialize.Executor

	exec.AsyncExec(func() {
		time.Sleep(time.Millisecond * 50)
		exec.AsyncExec(func() {
			time.Sleep(time.Millisecond * 50)
			exec.AsyncExec(func() {
				time.Sleep(time.Millisecond * 50)
			})
		})
	})

	for i := 0; i < 1e3; i++ {
		exec.AsyncExec(func() {})
	}

	<-exec.AsyncExec(func() {})
}

func TestASyncExec(t *testing.T) {
	exec := serialize.NewExecutor()
	count := 0
	completion0 := make(chan struct{})
	exec.AsyncExec(func() {
		assert.Equal(t, count, 0)
		count = 1
		close(completion0)
	})
	select {
	case <-completion0:
		assert.Fail(t, "exec.AsyncExec did run to completion before returning.")
	default:
	}
}

func TestSyncExec(t *testing.T) {
	var exec serialize.Executor
	count := 0
	completion0 := make(chan struct{})
	<-exec.AsyncExec(func() {
		assert.Equal(t, count, 0)
		count = 1
		close(completion0)
	})
	select {
	case <-completion0:
	default:
		assert.Fail(t, "exec.SyncExec did not run to completion before returning.")
	}
}

func TestExecOrder1(t *testing.T) {
	var exec serialize.Executor
	count := 0
	trigger0 := make(chan struct{})
	completion0 := make(chan struct{})
	exec.AsyncExec(func() {
		<-trigger0
		assert.Equal(t, count, 0)
		count = 1
		close(completion0)
	})
	trigger1 := make(chan struct{})
	completion1 := make(chan struct{})
	exec.AsyncExec(func() {
		<-trigger1
		assert.Equal(t, count, 1)
		count = 2
		close(completion1)
	})
	// Let the second one start first
	close(trigger1)
	close(trigger0)
	<-completion0
	<-completion1
}

// Same as TestExecOrder1 but making sure out of order fails as expected
func TestExecOrder2(t *testing.T) {
	var exec serialize.Executor
	count := 0

	trigger1 := make(chan struct{})
	completion1 := make(chan struct{})
	exec.AsyncExec(func() {
		<-trigger1
		assert.NotEqual(t, count, 1)
		count = 2
		close(completion1)
	})

	trigger0 := make(chan struct{})
	completion0 := make(chan struct{})
	exec.AsyncExec(func() {
		<-trigger0
		assert.NotEqual(t, count, 0)
		count = 1
		close(completion0)
	})
	// Let the second one start first
	close(trigger1)
	close(trigger0)
	<-completion0
	<-completion1
}

func TestExecOneAtATime(t *testing.T) {
	var exec serialize.Executor
	start := make(chan struct{})
	count := 100
	finished := make([]chan struct{}, count)
	var running int
	for i := 0; i < count; i++ {
		finished[i] = make(chan struct{})
		end := finished[i]
		exec.AsyncExec(func() {
			<-start
			assert.Equal(t, running, 0)
			running++
			assert.Equal(t, running, 1)
			running--
			close(end)
		})
	}
	close(start)
	for _, done := range finished {
		<-done
	}
}

func BenchmarkExecutorAsync(b *testing.B) {
	var exec serialize.Executor
	for i := 0; i < b.N-1; i++ {
		exec.AsyncExec(func() {})
	}
	<-exec.AsyncExec(func() {})
}

func BenchmarkExecutorSync(b *testing.B) {
	var exec serialize.Executor
	for i := 0; i < b.N; i++ {
		<-exec.AsyncExec(func() {})
	}
}
