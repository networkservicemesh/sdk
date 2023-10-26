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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/edwarnicke/serialize"
)

func TestAll(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	exec := &serialize.Executor{}
	t.Run("ASyncExec", func(t *testing.T) { ASyncExecTest(t, exec, 1) })
	t.Run("SyncExec", func(t *testing.T) { SyncExecTest(t, exec, 1) })
	t.Run("ExecOrder1", func(t *testing.T) { ExecOrder1Test(t, exec, 1000) })
	t.Run("ExecOrder2", func(t *testing.T) { ExecOrder2Test(t, exec, 1000) })
	t.Run("ExecOneAtATime", func(t *testing.T) { ExecOneAtATimeTest(t, exec, 1000) })
	t.Run("DataRace", func(t *testing.T) { DataRaceTest(t, exec, 1000) })
	t.Run("NestedAsyncDeadlock", func(t *testing.T) { NestedAsyncDeadlockTest(t, exec, 1000) })
	t.Run("ConcurrentAsyncAccess", func(t *testing.T) { ConcurrentAsyncMapAccessTest(t, exec, 1000) })
	t.Run("ConcurrentSyncAccess", func(t *testing.T) { ConcurrentSyncMapAccessTest(t, exec, 1000) })
}

func BenchmarkAll(b *testing.B) {
	defer goleak.VerifyNone(b, goleak.IgnoreCurrent())
	exec := &serialize.Executor{}
	b.Run("Performance", func(b *testing.B) {
		b.Run("ConcurrentAsyncMapAccess", func(b *testing.B) { ConcurrentAsyncMapAccessTest(b, exec, b.N) })
		b.Run("ConcurrentSyncMapAccessTest", func(b *testing.B) { ConcurrentSyncMapAccessTest(b, exec, b.N) })
		b.Run("ExecutorAsync", func(b *testing.B) { ExecutorAsyncBenchmark(b, exec, b.N) })
		b.Run("ExecutorSync", func(b *testing.B) { ExecutorSyncBenchmark(b, exec, b.N) })
	})
	b.Run("Tests", func(b *testing.B) {
		b.Run("ASyncExec", func(b *testing.B) { ASyncExecTest(b, exec, b.N) })
		b.Run("SyncExec", func(b *testing.B) { SyncExecTest(b, exec, b.N) })
		b.Run("ExecOrder1", func(b *testing.B) { ExecOrder1Test(b, exec, b.N) })
		b.Run("ExecOrder2", func(b *testing.B) { ExecOrder2Test(b, exec, b.N) })
		b.Run("ExecOneAtATime", func(b *testing.B) { ExecOneAtATimeTest(b, exec, b.N) })
		b.Run("DataRace", func(b *testing.B) { DataRaceTest(b, exec, b.N) })
		b.Run("NestedAsyncDeadlock", func(b *testing.B) { NestedAsyncDeadlockTest(b, exec, b.N) })
	})
	b.Run("Baselines", func(b *testing.B) {
		b.Run("BaselineConcurrentAccess", BaselineConcurrentAccess)
		b.Run("BaselineSyncMapReadWrite", BaselineSyncMapReadWrite)
	})
}

func DataRaceTest(t testing.TB, exec *serialize.Executor, count int) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	e := new(serialize.Executor)
	var arr []int
	wg := sync.WaitGroup{}
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(n int) {
			e.AsyncExec(func() {
				arr = append(arr, n)
				wg.Done()
			})
		}(i)
	}

	wg.Wait()
	if len(arr) != count {
		t.Errorf("len(arr): %d expected %d", len(arr), count)
	}
}

func ASyncExecTest(t testing.TB, exec *serialize.Executor, count int) {
	for i := 0; i < count; i++ {
		mu := sync.Mutex{}
		mu.Lock()
		done := make(chan struct{})
		exec.AsyncExec(func() {
			mu.Lock()
			close(done)
		})
		mu.Unlock()
		<-done
	}
}

func SyncExecTest(t testing.TB, exec *serialize.Executor, count int) {
	check := int32(0)
	for i := 0; i < count; i++ {
		j := int32(i)
		<-exec.AsyncExec(func() {
			atomic.AddInt32(&check, 1)
		})
		if atomic.LoadInt32(&check) == j {
			t.Error("exec.SyncExec did not run to completion before returning.")
		}
	}
}

func ExecOrder1Test(t testing.TB, exec *serialize.Executor, count int) {
	check := 0
	trigger := make([]chan struct{}, count)
	completion := make([]chan struct{}, len(trigger))
	for i := 0; i < count; i++ {
		j := i
		trigger[j] = make(chan struct{})
		completion[j] = make(chan struct{})
		exec.AsyncExec(func() {
			<-trigger[j]
			if check != j {
				t.Errorf("expected count == %d, actual %d", j, check)
			}
			check++
			close(completion[j])
		})
	}
	for i := len(trigger) - 1; i >= 0; i-- {
		close(trigger[i])
	}
	for _, c := range completion {
		select {
		case <-c:
		case <-time.After(500 * time.Millisecond):
			t.Errorf("timeout waiting for completion")
		}
	}
}

// Same as TestExecOrder1 but making sure out of order fails as expected
func ExecOrder2Test(t testing.TB, exec *serialize.Executor, count int) {
	for i := 0; i < count; i++ {
		defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
		check := 0

		trigger1 := make(chan struct{})
		completion1 := make(chan struct{})
		exec.AsyncExec(func() {
			<-trigger1
			if !(check != 1) {
				t.Errorf("expected count != 1, actual %d", check)
			}
			check = 2
			close(completion1)
		})

		trigger0 := make(chan struct{})
		completion0 := make(chan struct{})
		exec.AsyncExec(func() {
			<-trigger0
			if !(check != 0) {
				t.Errorf("expected count != 0, actual %d", check)
			}
			check = 1
			close(completion0)
		})
		// Let the second one start first
		close(trigger1)
		close(trigger0)
		<-completion0
		<-completion1
	}
}

func ExecOneAtATimeTest(t testing.TB, exec *serialize.Executor, count int) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	start := make(chan struct{})
	finished := make([]chan struct{}, count)
	var running int
	for i := 0; i < count; i++ {
		finished[i] = make(chan struct{})
		end := finished[i]
		exec.AsyncExec(func() {
			<-start
			if running != 0 {
				t.Errorf("expected running == 0, actual %d", count)
			}
			running++
			if running != 1 {
				t.Errorf("expected running == 1, actual %d", count)
			}
			running--
			close(end)
		})
	}
	close(start)
	for _, done := range finished {
		<-done
	}
}

func NestedAsyncDeadlockTest(t testing.TB, exec *serialize.Executor, count int) {
	startCh := make(chan struct{})
	exec.AsyncExec(func() {
		<-startCh
		exec.AsyncExec(func() {
			<-startCh
			exec.AsyncExec(func() {
				<-startCh
			})
		})
	})
	for i := 0; i < count; i++ {
		exec.AsyncExec(func() {})
	}
	close(startCh)
	<-exec.AsyncExec(func() {})
}

func ConcurrentAsyncMapAccessTest(t testing.TB, exec *serialize.Executor, count int) {
	ConcurrentAccessTest(t, exec, count, false)
}

func ConcurrentSyncMapAccessTest(t testing.TB, exec *serialize.Executor, count int) {
	ConcurrentAccessTest(t, exec, count, true)
}

func ConcurrentAccessTest(t testing.TB, exec *serialize.Executor, count int, synchronous bool) {
	m := make(map[int]bool)
	for i := 0; i < count; i++ {
		j := i
		done := exec.AsyncExec(func() {
			if j > 0 && !m[j-1] {
				t.Errorf("did not find expected value in map[%d]bool", j)
			}
			m[j] = true
		})
		if synchronous {
			<-done
		}
	}
	<-exec.AsyncExec(func() {})
}

func BaselineConcurrentAccess(b *testing.B) {
	m := make(map[int]bool)
	for i := 0; i < b.N; i++ {
		j := i
		if j > 0 && !m[j-1] {
			b.Errorf("did not find expected value in map[%d]bool", j)
		}
		m[j] = true
	}
}

func BaselineSyncMapReadWrite(b *testing.B) {
	m := sync.Map{}
	for i := 0; i < b.N; i++ {
		j := i
		_, ok := m.Load(j - 1)
		if j > 0 && !ok {
			b.Errorf("did not find expected value in map[%d]bool", j)
		}
		m.Store(j, true)
	}
}

func ExecutorAsyncBenchmark(b testing.TB, exec *serialize.Executor, count int) {
	for i := 0; i < count-1; i++ {
		exec.AsyncExec(func() {})
	}
	<-exec.AsyncExec(func() {})
}

func ExecutorSyncBenchmark(b testing.TB, exec *serialize.Executor, count int) {
	for i := 0; i < count; i++ {
		<-exec.AsyncExec(func() {})
	}
}
