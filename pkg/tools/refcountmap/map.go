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

package refcountmap

import (
	"sync"
)

type refcountEntry struct {
	count int
	value interface{}
}

// Map is a map[string]interface{} wrapped with refcounting. Each Store, Load, LoadOrStore increments the count, each
// LoadAndDelete, Delete decrements the count. When the count == 1, LoadAndDelete, Delete deletes.
type Map struct {
	m    map[string]*refcountEntry
	once sync.Once
}

func (m *Map) init() {
	m.once.Do(func() {
		m.m = make(map[string]*refcountEntry)
	})
}

// Store sets the value for a key
// count = 1.
func (m *Map) Store(key string, value interface{}) {
	m.init()

	m.m[key] = &refcountEntry{
		count: 1,
		value: value,
	}
}

// LoadOrStore returns the existing value for the key if present. Otherwise, it stores and returns the given value. The
// loaded result is true if the value was loaded, false if stored.
// store  -->  count = 1
// load   -->  count += 1
func (m *Map) LoadOrStore(key string, value interface{}) (interface{}, bool) {
	m.init()

	entry, loaded := m.m[key]
	if !loaded {
		entry = &refcountEntry{
			value: value,
		}
		m.m[key] = entry
	}
	entry.count++

	return entry.value, loaded
}

// Load returns the value stored in the map for a key, or nil if no value is present. The loaded result indicates
// whether value was found in the map.
// count += 1
func (m *Map) Load(key string) (interface{}, bool) {
	m.init()

	entry, loaded := m.m[key]
	if !loaded {
		return nil, false
	}
	entry.count++

	return entry.value, loaded
}

// LoadUnsafe returns the value stored in the map for a key, or nil if no value is present. The loaded result indicates
// whether value was found in the map.
func (m *Map) LoadUnsafe(key string) (interface{}, bool) {
	m.init()

	entry, loaded := m.m[key]
	if !loaded {
		return nil, false
	}

	return entry.value, loaded
}

// LoadAndDelete is trying to delete the value for a key, returning the previous value if any. The loaded result
// reports whether the key was present.
// count == 1  -->  delete
// count > 1   -->  count -=1
func (m *Map) LoadAndDelete(key string) (interface{}, bool) {
	m.init()

	entry, loaded := m.m[key]
	if !loaded {
		return nil, false
	}

	switch entry.count {
	case 1:
		delete(m.m, key)
	default:
		entry.count--
	}

	return entry.value, true
}

// Delete is trying to delete the value for a key.
// count == 1  -->  delete
// count > 1   -->  count -=1
func (m *Map) Delete(key string) {
	m.LoadAndDelete(key)
}
