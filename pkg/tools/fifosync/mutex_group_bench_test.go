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

type Map struct {
	init      sync.Once
	executor  serialize.Executor
	executors map[string]*executor
}

type executor struct {
	executor serialize.Executor
	count    int
}

func (m *Map) AsyncExec(id string, f func()) <-chan struct{} {
	m.init.Do(func() {
		m.executors = map[string]*executor{}
	})

	var ex *executor
	<-m.executor.AsyncExec(func() {
		var ok bool
		ex, ok = m.executors[id]
		if !ok {
			ex = &executor{}
			m.executors[id] = ex
		}
		ex.count++
	})

	return ex.executor.AsyncExec(func() {
		f()
		m.executor.AsyncExec(func() {
			ex.count--
		})
	})
}

func (m *Map) Delete(id string) bool {
	var isDeleted bool
	<-m.executor.AsyncExec(func() {
		ex, ok := m.executors[id]
		if !ok || ex.count == 0 {
			delete(m.executors, id)
			isDeleted = true
			return
		}
	})
	return isDeleted
}

var ids = []string{
	"00", "01", "02", "03", "04", "05", "06", "07", "08", "09",
	"10", "11", "12", "13", "14", "15", "16", "17", "18", "19",
	"20", "21", "22", "23", "24", "25", "26", "27", "28", "29",
	"30", "31", "32", "33", "34", "35", "36", "37", "38", "39",
	"40", "41", "42", "43", "44", "45", "46", "47", "48", "49",
	"50", "51", "52", "53", "54", "55", "56", "57", "58", "59",
	"60", "61", "62", "63", "64", "65", "66", "67", "68", "69",
	"70", "71", "72", "73", "74", "75", "76", "77", "78", "79",
	"80", "81", "82", "83", "84", "85", "86", "87", "88", "89",
	"90", "91", "92", "93", "94", "95", "96", "97", "98", "99",
}

func BenchmarkMutexGroup_Lock_Same(b *testing.B) {
	var g fifosync.MutexGroup
	for i := 0; i < b.N; i++ {
		id := ids[0]
		g.Lock(id)
		g.Unlock(id)
	}
	g.Delete(ids[0])
}

func BenchmarkMap_AsyncExec_Same(b *testing.B) {
	var m Map
	for i := 0; i < b.N; i++ {
		id := ids[0]
		m.AsyncExec(id, func() {})
	}
	m.Delete(ids[0])
}

func BenchmarkMutexGroup_Lock_Different(b *testing.B) {
	var g fifosync.MutexGroup
	for i := 0; i < b.N; i++ {
		id := ids[i%len(ids)]
		g.Lock(id)
		g.Unlock(id)
	}
	for i := 0; i < 100; i++ {
		g.Delete(ids[i])
	}
}

func BenchmarkMap_AsyncExec_Different(b *testing.B) {
	var m Map
	for i := 0; i < b.N; i++ {
		id := ids[i%len(ids)]
		m.AsyncExec(id, func() {})
	}
	for i := 0; i < 100; i++ {
		m.Delete(ids[i])
	}
}

func BenchmarkMutexGroup_Lock_Same_Goroutine(b *testing.B) {
	var g fifosync.MutexGroup
	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := ids[0]
			g.Lock(id)
			g.Unlock(id)
		}
	})
	g.Delete(ids[0])
}

func BenchmarkMap_AsyncExec_Same_Goroutine(b *testing.B) {
	var m Map
	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := ids[0]
			m.AsyncExec(id, func() {})
		}
	})
	m.Delete(ids[0])
}

func BenchmarkMutexGroup_Lock_Different_Goroutine(b *testing.B) {
	var g fifosync.MutexGroup
	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			id := ids[i%len(ids)]
			g.Lock(id)
			g.Unlock(id)
		}
	})
	for i := 0; i < 100; i++ {
		g.Delete(ids[i])
	}
}

func BenchmarkMap_AsyncExec_Different_Goroutine(b *testing.B) {
	var m Map
	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			id := ids[i%len(ids)]
			m.AsyncExec(id, func() {})
		}
	})
	for i := 0; i < 100; i++ {
		m.Delete(ids[i])
	}
}
