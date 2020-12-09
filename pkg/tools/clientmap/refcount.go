// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
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

package clientmap

import (
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type entry struct {
	count int
	lock  sync.Mutex
	value networkservice.NetworkServiceClient
}

// RefcountMap is a sync.Map wrapped with refcounting. Each Store, Load, LoadOrStore increments the count, each
// LoadAndDelete, Delete decrements the count. When the count == 1, LoadAndDelete, Delete deletes.
type RefcountMap struct {
	m sync.Map
}

// Store sets the value for a key
// count = 1.
func (m *RefcountMap) Store(key string, newValue networkservice.NetworkServiceClient) {
	m.m.Store(key, &entry{
		count: 1,
		value: newValue,
	})
}

// LoadOrStore returns the existing value for the key if present. Otherwise, it stores and returns the given value. The
// loaded result is true if the value was loaded, false if stored.
// store  -->  count = 1
// load   -->  count += 1
func (m *RefcountMap) LoadOrStore(key string, newValue networkservice.NetworkServiceClient) (value networkservice.NetworkServiceClient, loaded bool) {
	var raw interface{}
	raw, loaded = m.m.LoadOrStore(key, &entry{
		count: 1,
		value: newValue,
	})
	entry := raw.(*entry)
	if !loaded {
		return entry.value, false
	}

	entry.lock.Lock()

	if entry.count == 0 {
		entry.lock.Unlock()
		return m.LoadOrStore(key, newValue)
	}
	entry.count++

	entry.lock.Unlock()

	return entry.value, true
}

// Load returns the value stored in the map for a key, or nil if no value is present. The loaded result indicates
// whether value was found in the map.
// count += 1
func (m *RefcountMap) Load(key string) (value networkservice.NetworkServiceClient, loaded bool) {
	var raw interface{}
	raw, loaded = m.m.Load(key)
	if !loaded {
		return nil, false
	}
	entry := raw.(*entry)

	entry.lock.Lock()
	defer entry.lock.Unlock()

	if entry.count == 0 {
		return nil, false
	}
	entry.count++

	return entry.value, true
}

// LoadAndDelete is trying to delete the value for a key, returning the previous value if any. The loaded result
// reports whether the key was present.
// count == 1  -->  delete
// count > 1   -->  count -= 1
func (m *RefcountMap) LoadAndDelete(key string) (value networkservice.NetworkServiceClient, loaded, deleted bool) {
	var raw interface{}
	raw, loaded = m.m.Load(key)
	if !loaded {
		return nil, false, true
	}
	entry := raw.(*entry)

	entry.lock.Lock()
	defer entry.lock.Unlock()

	if entry.count == 0 {
		return nil, false, true
	}
	entry.count--

	if entry.count == 0 {
		m.m.Delete(key)
		deleted = true
	}

	return entry.value, true, deleted
}

// Delete is trying to delete the value for a key.
// count == 1  -->  delete
// count > 1   -->  count -= 1
func (m *RefcountMap) Delete(key string) (deleted bool) {
	_, _, deleted = m.LoadAndDelete(key)
	return deleted
}
