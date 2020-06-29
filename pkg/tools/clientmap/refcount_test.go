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

package clientmap_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/null"
	"github.com/networkservicemesh/sdk/pkg/tools/clientmap"
)

func TestRefcountMap(t *testing.T) {
	m := &clientmap.RefcountMap{}
	client := null.NewClient()
	key := "key"

	m.Store(key, client)    // refcount == 1
	client, _ = m.Load(key) // refcount == 2
	assert.Equal(t, client, client)
	m.Delete(key) // refcount == 1
	m.Delete(key) // refcount == 0
	CheckRefcount(t, m, key, 0)

	m.Store(key, client) // refcount == 1
	m.Store(key, client) // refcount == 1
	CheckRefcount(t, m, key, 1)
	m.Delete(key) // refcount == 1
	CheckRefcount(t, m, key, 0)

	for i := 0; i < 10; i++ {
		_, loaded := m.LoadOrStore(key, client) // refcount == i
		assert.True(t, loaded || i == 0)
	}
	CheckRefcount(t, m, key, 10)
	CheckRefcount(t, m, key, 10)

	// Check store resets counter
	m.Store(key, client)
	CheckRefcount(t, m, key, 1)
	m.Delete(key)
	CheckRefcount(t, m, key, 0)
}

func CheckRefcount(t *testing.T, m *clientmap.RefcountMap, key string, refcount int) {
	var loaded bool
	client, found := m.Load(key) // refcount + 1
	m.Delete(key)                // refcount
	if !found {
		assert.Equal(t, 0, refcount)
		return
	}

	// Drain out the refcount
	for i := refcount; i > 0; i-- {
		m.Delete(key)                               // refcount == i-1
		client, loaded = m.LoadOrStore(key, client) // refcount == i
		assert.True(t, loaded || i == 1)
		m.Delete(key) // refcount == i-1
	}

	// Replace th refcount
	for i := 0; i < refcount; i++ {
		client, loaded = m.LoadOrStore(key, client)
		assert.True(t, loaded || i == 0)
	}
}
