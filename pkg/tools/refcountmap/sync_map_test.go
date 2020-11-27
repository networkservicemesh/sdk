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
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/tools/refcountmap"
)

const (
	parallelCount = 1000
)

type sample struct {
	name      string
	withValue bool
	actions   []action
	validator func(count int, results []bool) bool
}

var samples = []*sample{
	{
		name: "LoadOrStore + LoadOrStore",
		// 0 1 -> 2 { false, true }
		// 1 0 -> 2 { true, false }
		actions: []action{loadOrStore, loadOrStore},
		validator: func(count int, results []bool) bool {
			return (count == 2) && !results[0] && results[1] ||
				(count == 2) && results[0] && !results[1]
		},
	},
	{
		name: "LoadOrStore + LoadAndDelete",
		// 0 1 -> 0 { false, true }
		// 1 0 -> 1 { false, false }
		actions: []action{loadOrStore, loadAndDelete},
		validator: func(count int, results []bool) bool {
			return (count == 0) && !results[0] && results[1] ||
				(count == 1) && !results[0] && !results[1]
		},
	},
	{
		name:      "LoadOrStore + LoadAndDelete with value",
		withValue: true,
		// 0 1 -> 1 { true, true }
		// 1 0 -> 1 { false, true }
		actions: []action{loadOrStore, loadAndDelete},
		validator: func(count int, results []bool) bool {
			return (count == 1) && results[0] && results[1] ||
				(count == 1) && !results[0] && results[1]
		},
	},
	{
		name:      "LoadAndDelete + LoadAndDelete",
		withValue: true,
		// 0 1 -> 0 { true, false }
		// 1 0 -> 0 { false, true }
		actions: []action{loadAndDelete, loadAndDelete},
		validator: func(count int, results []bool) bool {
			return (count == 0) && results[0] && !results[1] ||
				(count == 0) && !results[0] && results[1]
		},
	},
	{
		name:      "LoadAndDelete + Load",
		withValue: true,
		// 0 1 -> 0 { true, false }
		// 1 0 -> 1 { true, true }
		actions: []action{loadAndDelete, load},
		validator: func(count int, results []bool) bool {
			return (count == 0) && results[0] && !results[1] ||
				(count == 1) && results[0] && results[1]
		},
	},
	{
		name:      "LoadAndDelete + LoadAndDelete + Load",
		withValue: true,
		// 0 1 2 -> 0 { true, false, false }
		// 0 2 1 -> 0 { true, false, false } (same as ^)
		// 1 0 2 -> 0 { false, true, false }
		// 1 2 0 -> 0 { true, true, true }
		// 2 0 1 -> 0 { false, true, false } (same as ^^)
		// 2 1 0 -> 0 { true, true, true } (same as ^^)
		actions: []action{loadAndDelete, loadAndDelete, load},
		validator: func(count int, results []bool) bool {
			return (count == 0) && results[0] && !results[1] && !results[2] ||
				(count == 0) && !results[0] && results[1] && !results[2] ||
				(count == 0) && results[0] && results[1] && results[2]
		},
	},
	{
		name:      "LoadOrStore + LoadOrStore + LoadAndDelete",
		withValue: true,
		// 0 1 2 -> 2 { true, true, true }
		// 0 2 1 -> 2 { true, true, true } (same as ^)
		// 1 0 2 -> 2 { true, true, true } (same as ^)
		// 1 2 0 -> 2 { false, true, true }
		// 2 0 1 -> 2 { true, true, true } (same as ^^)
		// 2 1 0 -> 2 { true, false, true }
		actions: []action{loadOrStore, loadOrStore, loadAndDelete},
		validator: func(count int, results []bool) bool {
			return (count == 2) && results[0] && results[1] && results[2] ||
				(count == 2) && !results[0] && results[1] && results[2] ||
				(count == 2) && results[0] && !results[1] && results[2]
		},
	},
}

func TestSyncMap(t *testing.T) {
	for _, sample := range samples {
		runSample(t, new(refcountmap.SyncMap), sample)
	}
}

func runSample(t *testing.T, m *refcountmap.SyncMap, sample *sample) {
	t.Run(sample.name, func(t *testing.T) {
		value := map[string]string{"a": "A"}
		for i := 0; i < parallelCount; i++ {
			key := strconv.Itoa(i)
			if sample.withValue {
				m.Store(key, value)
			}

			results := runActions(m, key, value, sample.actions...)

			var count int
			for _, loaded := m.LoadAndDelete(key); loaded; _, loaded = m.LoadAndDelete(key) {
				count++
			}

			assert.True(t, sample.validator(count, results),
				"count = %v, results = %v", count, results)
		}
	})
}

type action func(*refcountmap.SyncMap, string, interface{}) bool

func loadOrStore(m *refcountmap.SyncMap, key string, value interface{}) bool {
	_, ok := m.LoadOrStore(key, value)
	return ok
}

func load(m *refcountmap.SyncMap, key string, _ interface{}) bool {
	_, ok := m.Load(key)
	return ok
}

func loadAndDelete(m *refcountmap.SyncMap, key string, _ interface{}) bool {
	_, ok := m.LoadAndDelete(key)
	return ok
}

func runActions(m *refcountmap.SyncMap, key string, value interface{}, actions ...action) []bool {
	wg := new(sync.WaitGroup)
	wg.Add(len(actions))

	results := make([]bool, len(actions))
	for i := range actions {
		go func(k int) {
			results[k] = actions[k](m, key, value)
			wg.Done()
		}(i)
	}

	wg.Wait()

	return results
}
