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

package refcountmap_test

import (
	"sync"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/tools/refcountmap"
)

type lockedMap struct {
	m    refcountmap.Map
	lock sync.Mutex
}

func (m *lockedMap) Store(key string, value interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.m.Store(key, value)
}

func (m *lockedMap) LoadOrStore(key string, value interface{}) (interface{}, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.m.LoadOrStore(key, value)
}

func (m *lockedMap) Load(key string) (interface{}, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.m.Load(key)
}

func (m *lockedMap) LoadUnsafe(key string) (interface{}, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.m.LoadUnsafe(key)
}

func (m *lockedMap) LoadAndDelete(key string) (interface{}, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.m.LoadAndDelete(key)
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
	"90", "91", "92", "93", "94", "95", "96", "97", "98", "99", "100",
}

func BenchmarkSyncMap(b *testing.B) {
	var m sync.Map
	b.SetParallelism(1000)
	b.RunParallel(func(pb *testing.PB) {
		var value interface{} = map[string]string{"a": "A"}
		for i := 0; pb.Next(); i++ {
			id := ids[i%len(ids)]
			switch i % 7 {
			case 0:
				m.Store(id, value)
			case 1:
				value, _ = m.LoadOrStore(id, value)
			case 2:
				value, _ = m.Load(id)
			case 4, 5, 6:
				value, _ = m.LoadAndDelete(id)
			}
		}
	})
}

func BenchmarkRefcountSyncMap(b *testing.B) {
	var m refcountmap.SyncMap
	b.SetParallelism(1000)
	b.RunParallel(func(pb *testing.PB) {
		var value interface{} = map[string]string{"a": "A"}
		for i := 0; pb.Next(); i++ {
			id := ids[i%len(ids)]
			switch i % 8 {
			case 0:
				m.Store(id, value)
			case 1:
				value, _ = m.LoadOrStore(id, value)
			case 2:
				value, _ = m.Load(id)
			case 3:
				value, _ = m.LoadUnsafe(id)
			case 4, 5, 6, 7:
				value, _ = m.LoadAndDelete(id)
			}
		}
	})
}

func BenchmarkRefcountLockedMap(b *testing.B) {
	var m lockedMap
	b.SetParallelism(1000)
	b.RunParallel(func(pb *testing.PB) {
		var value interface{} = map[string]string{"a": "A"}
		for i := 0; pb.Next(); i++ {
			id := ids[i%len(ids)]
			switch i % 8 {
			case 0:
				m.Store(id, value)
			case 1:
				value, _ = m.LoadOrStore(id, value)
			case 2:
				value, _ = m.Load(id)
			case 3:
				value, _ = m.LoadUnsafe(id)
			case 4, 5, 6, 7:
				value, _ = m.LoadAndDelete(id)
			}
		}
	})
}
