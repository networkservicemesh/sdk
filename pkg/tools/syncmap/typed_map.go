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

package syncmap

import (
	"sync"

	"github.com/cheekybits/genny/generic"
)

// V is an type of values for syncmap
type V generic.Type

// K is an type for keys for syncmap
type K generic.Type

// KVMap is like a Go map[K]*V{} but is safe for concurrent use
// by multiple goroutines without additional locking or coordination and type casting.
type KVMap struct {
	m sync.Map
}

// Load returns the value stored in the map for a key, or nil if no
// value is present.
// The ok result indicates whether value was found in the map.
func (m *KVMap) Load(k K) (*V, bool) {
	v, ok := m.m.Load(k)
	if ok {
		return v.(*V), ok
	}
	return nil, false
}

// Store sets the value for a key.
func (m *KVMap) Store(k K, v *V) {
	m.m.Store(k, v)
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (m *KVMap) LoadOrStore(k K, v *V) (*V, bool) {
	val, ok := m.m.LoadOrStore(k, v)
	if ok {
		return val.(*V), ok
	}
	return nil, false
}

// Delete deletes the value for a key.
func (m *KVMap) Delete(k K) {
	m.m.Delete(k)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
//
// Range does not necessarily correspond to any consistent snapshot of the Map's
// contents: no key will be visited more than once, but if the value for any key
// is stored or deleted concurrently, Range may reflect any mapping for that key
// from any point during the Range call.
//
// Range may be O(N) with the number of elements in the map even if f returns
// false after a constant number of calls.
func (m *KVMap) Range(f func(k K, v *V) bool) {
	m.m.Range(func(key, value interface{}) bool {
		return f(key.(K), value.(*V))
	})
}

// LoadAll loads all stored values in the map.
func (m *KVMap) LoadAll() []*V {
	var all []*V
	m.Range(func(_ K, v *V) bool {
		all = append(all, v)
		return true
	})
	return all
}
