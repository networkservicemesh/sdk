// Copyright (c) 2024 Cisco and/or its affiliates.
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

	"github.com/edwarnicke/genericsync"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

type nsCache struct {
	expireTimeout time.Duration
	entries       genericsync.Map[string, *cacheEntry[registry.NetworkService]]
	clockTime     clock.Clock
}

func newNSCache(ctx context.Context, opts ...NSCacheOption) *nsCache {
	c := &nsCache{
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
				c.entries.Range(func(_ string, e *cacheEntry[registry.NetworkService]) bool {
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

func (c *nsCache) LoadOrStore(value *registry.NetworkService, cancel context.CancelFunc) (*cacheEntry[registry.NetworkService], bool) {
	var once sync.Once

	entry, ok := c.entries.LoadOrStore(value.GetName(), &cacheEntry[registry.NetworkService]{
		value:          value,
		expirationTime: c.clockTime.Now().Add(c.expireTimeout),
		cleanup: func() {
			once.Do(func() {
				c.entries.Delete(value.GetName())
				cancel()
			})
		}})

	return entry, ok
}

func (c *nsCache) Load(ctx context.Context, query *registry.NetworkService) *registry.NetworkService {
	entry, ok := c.entries.Load(query.Name)
	if ok {
		entry.lock.Lock()
		defer entry.lock.Unlock()
		if c.clockTime.Until(entry.expirationTime) < 0 {
			entry.cleanup()
		} else {
			entry.expirationTime = c.clockTime.Now().Add(c.expireTimeout)
			return entry.value
		}
	}

	return nil
}

type cacheEntry[T registry.NetworkService | registry.NetworkServiceEndpoint] struct {
	value          *T
	expirationTime time.Time
	lock           sync.Mutex
	cleanup        func()
}

func (e *cacheEntry[T]) Update(value *T) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.value = value
}

func (e *cacheEntry[_]) Cleanup() {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.cleanup()
}
