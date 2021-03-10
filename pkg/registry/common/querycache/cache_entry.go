// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package querycache

import (
	"sync"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type cacheEntry struct {
	nse            *registry.NetworkServiceEndpoint
	expireTimeout  time.Duration
	expirationTime time.Time
	timer          *time.Timer
	lock           sync.Mutex
	cleanup        func()
}

func newCacheEntry(nse *registry.NetworkServiceEndpoint, expireTimeout time.Duration, cleanup func()) *cacheEntry {
	var once sync.Once
	e := &cacheEntry{
		nse:            nse,
		expireTimeout:  expireTimeout,
		expirationTime: time.Now().Add(expireTimeout),
		cleanup: func() {
			once.Do(cleanup)
		},
	}

	e.timer = time.AfterFunc(expireTimeout, func() {
		e.lock.Lock()
		defer e.lock.Unlock()

		if e.expirationTime.After(time.Now()) {
			e.timer.Reset(time.Until(e.expirationTime))
			return
		}

		e.cleanup()
	})

	return e
}

func (e *cacheEntry) Read() (*registry.NetworkServiceEndpoint, bool) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.expirationTime.Before(time.Now()) {
		e.cleanup()
		return nil, false
	}

	e.expirationTime = time.Now().Add(e.expireTimeout)

	return e.nse, true
}

func (e *cacheEntry) Update(nse *registry.NetworkServiceEndpoint) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.nse = nse
}

func (e *cacheEntry) Cleanup() {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.expirationTime = time.Time{}

	if e.timer.Stop() {
		e.cleanup()
	}
}
