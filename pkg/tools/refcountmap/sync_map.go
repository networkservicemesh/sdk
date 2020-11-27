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

type syncRefcountEntry struct {
	count int
	value interface{}
	lock  sync.Mutex
}

// SyncMap is a sync.Map wrapped with refcounting. Each Store, Load, LoadOrStore increments the count, each LoadAndDelete,
// Delete decrements the count. When the count == 1, LoadAndDelete, Delete deletes.
type SyncMap struct {
	m sync.Map
}

// Store sets the value for a key
// count = 1.
func (m *SyncMap) Store(key string, value interface{}) {
	m.m.Store(key, &syncRefcountEntry{
		count: 1,
		value: value,
	})
}

// LoadOrStore returns the existing value for the key if present. Otherwise, it stores and returns the given value. The
// loaded result is true if the value was loaded, false if stored.
// store  -->  count = 1
// load   -->  count += 1
func (m *SyncMap) LoadOrStore(key string, value interface{}) (interface{}, bool) {
	raw, loaded := m.m.LoadOrStore(key, &syncRefcountEntry{
		count: 1,
		value: value,
	})
	entry := raw.(*syncRefcountEntry)
	if !loaded {
		return entry.value, false
	}

	entry.lock.Lock()

	if entry.count <= 0 {
		entry.lock.Unlock()
		return m.LoadOrStore(key, value)
	}
	entry.count++

	entry.lock.Unlock()

	return entry.value, true
}

// Load returns the value stored in the map for a key, or nil if no value is present. The loaded result indicates
// whether value was found in the map.
// count += 1
func (m *SyncMap) Load(key string) (interface{}, bool) {
	raw, loaded := m.m.Load(key)
	if !loaded {
		return nil, false
	}
	entry := raw.(*syncRefcountEntry)

	entry.lock.Lock()

	if entry.count <= 0 {
		entry.lock.Unlock()
		return nil, false
	}
	entry.count++

	entry.lock.Unlock()

	return entry.value, loaded
}

// LoadUnsafe returns the value stored in the map for a key, or nil if no value is present. The loaded result indicates
// whether value was found in the map.
func (m *SyncMap) LoadUnsafe(key string) (interface{}, bool) {
	raw, loaded := m.m.Load(key)
	if !loaded {
		return nil, false
	}
	entry := raw.(*syncRefcountEntry)

	return entry.value, loaded
}

// LoadAndDelete is trying to delete the value for a key, returning the previous value if any. The loaded result
// reports whether the key was present.
// count == 1  -->  delete
// count > 1   -->  count -=1
func (m *SyncMap) LoadAndDelete(key string) (interface{}, bool) {
	raw, loaded := m.m.Load(key)
	if !loaded {
		return nil, false
	}
	entry := raw.(*syncRefcountEntry)

	entry.lock.Lock()
	defer entry.lock.Unlock()

	switch entry.count {
	case 0:
		return nil, false
	case 1:
		m.m.Delete(key)
		fallthrough
	default:
		entry.count--
		return entry.value, true
	}
}

// Delete is trying to delete the value for a key.
// count == 1  -->  delete
// count > 1   -->  count -=1
func (m *SyncMap) Delete(key string) {
	m.LoadAndDelete(key)
}
