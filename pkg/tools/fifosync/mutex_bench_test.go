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

package fifosync_test

import (
	"sync"
	"testing"

	"github.com/edwarnicke/serialize"

	"github.com/networkservicemesh/sdk/pkg/tools/fifosync"
)

func BenchmarkMutex_Lock(b *testing.B) {
	var m sync.Mutex
	for i := 0; i < b.N; i++ {
		m.Lock()
		m.Unlock()
	}
}

func BenchmarkChannelMutex_Lock(b *testing.B) {
	var m fifosync.ChannelMutex
	for i := 0; i < b.N; i++ {
		m.Lock()
		m.Unlock()
	}
}

func BenchmarkFIFOMutex_Lock(b *testing.B) {
	var m fifosync.Mutex
	for i := 0; i < b.N; i++ {
		m.Lock()
		m.Unlock()
	}
}

func BenchmarkNaiveMutex_Lock(b *testing.B) {
	var m fifosync.NaiveMutex
	for i := 0; i < b.N; i++ {
		m.Lock()
		m.Unlock()
	}
}

func BenchmarkExecutor_AsyncExec(b *testing.B) {
	var e serialize.Executor
	for i := 0; i < b.N; i++ {
		<-e.AsyncExec(func() {})
	}
}

func BenchmarkMutex_Lock_Goroutine(b *testing.B) {
	var m sync.Mutex
	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.Lock()
			m.Unlock()
		}
	})
}

func BenchmarkChannelMutex_Lock_Goroutine(b *testing.B) {
	var m fifosync.ChannelMutex
	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.Lock()
			m.Unlock()
		}
	})
}

func BenchmarkFIFOMutex_Lock_Goroutine(b *testing.B) {
	var m fifosync.Mutex
	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.Lock()
			m.Unlock()
		}
	})
}

func BenchmarkNaiveMutex_Lock_Goroutine(b *testing.B) {
	var m fifosync.NaiveMutex
	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.Lock()
			m.Unlock()
		}
	})
}

func BenchmarkExecutor_AsyncExec_Goroutine(b *testing.B) {
	var e serialize.Executor
	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			<-e.AsyncExec(func() {})
		}
	})
}
