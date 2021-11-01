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
	"context"
	"sync"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

type cache struct {
	expireTimeout time.Duration
	entries       cacheEntryMap
	clockTime     clock.Clock
}

func newCache(ctx context.Context, opts ...Option) *cache {
	c := &cache{
		expireTimeout: time.Minute,
		clockTime:     clock.FromContext(ctx),
	}

	for _, opt := range opts {
		opt(c)
	}

	ticker := c.clockTime.Ticker(c.expireTimeout)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C():
				c.entries.Range(func(_ string, e *cacheEntry) bool {
					e.lock.Lock()
					defer e.lock.Unlock()

					if c.clockTime.Until(e.expirationTime) < 0 {
						e.cleanup()
					}

					return true
				})
			}
		}
	}()

	return c
}

func (c *cache) LoadOrStore(key string, nse *registry.NetworkServiceEndpoint, cancel context.CancelFunc) (*cacheEntry, bool) {
	var once sync.Once
	return c.entries.LoadOrStore(key, &cacheEntry{
		nse:            nse,
		expirationTime: c.clockTime.Now().Add(c.expireTimeout),
		cleanup: func() {
			once.Do(func() {
				c.entries.Delete(key)
				cancel()
			})
		},
	})
}

func (c *cache) Load(key string) (*registry.NetworkServiceEndpoint, bool) {
	e, ok := c.entries.Load(key)
	if !ok {
		return nil, false
	}

	e.lock.Lock()
	defer e.lock.Unlock()

	if c.clockTime.Until(e.expirationTime) < 0 {
		e.cleanup()
		return nil, false
	}

	e.expirationTime = c.clockTime.Now().Add(c.expireTimeout)

	return e.nse, true
}

type cacheEntry struct {
	nse            *registry.NetworkServiceEndpoint
	expirationTime time.Time
	lock           sync.Mutex
	cleanup        func()
}

func (e *cacheEntry) Update(nser *registry.NetworkServiceEndpointResponse) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.nse = nser.NetworkServiceEndpoint
}

func (e *cacheEntry) Cleanup() {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.cleanup()
}
