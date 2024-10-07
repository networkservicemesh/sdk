// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type nseCache struct {
	expireTimeout time.Duration
	entries       genericsync.Map[string, *cacheEntry[registry.NetworkServiceEndpoint]]
	clockTime     clock.Clock
}

func newNSECache(ctx context.Context, opts ...Option) *nseCache {
	c := &nseCache{
		expireTimeout: time.Minute,
		clockTime:     clock.FromContext(ctx),
	}

	// for _, opt := range opts {
	// 	opt(c)
	// }

	ticker := c.clockTime.Ticker(c.expireTimeout)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C():
				c.entries.Range(func(_ string, e *cacheEntry[registry.NetworkServiceEndpoint]) bool {
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

func (c *nseCache) LoadOrStore(value *registry.NetworkServiceEndpoint, cancel context.CancelFunc) (*cacheEntry[registry.NetworkServiceEndpoint], bool) {
	var once sync.Once

	entry, ok := c.entries.LoadOrStore(value.GetName(), &cacheEntry[registry.NetworkServiceEndpoint]{
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

func (c *nseCache) add(entry *cacheEntry[registry.NetworkServiceEndpoint], values []*registry.NetworkServiceEndpoint) []*registry.NetworkServiceEndpoint {
	entry.lock.Lock()
	defer entry.lock.Unlock()
	if c.clockTime.Until(entry.expirationTime) < 0 {
		entry.cleanup()
	} else {
		entry.expirationTime = c.clockTime.Now().Add(c.expireTimeout)
		values = append(values, entry.value)
	}

	return values
}

// Checks if a is a subset of b
func subset(a, b []string) bool {
	set := make(map[string]struct{})
	for _, value := range a {
		set[value] = struct{}{}
	}

	for _, value := range b {
		if _, found := set[value]; !found {
			return false
		}
	}

	return true
}

func (c *nseCache) Load(ctx context.Context, query *registry.NetworkServiceEndpointQuery) []*registry.NetworkServiceEndpoint {
	values := make([]*registry.NetworkServiceEndpoint, 0)

	log.FromContext(ctx).WithField("time", time.Now()).Infof("query: %v\n", query)

	if query.NetworkServiceEndpoint.Name != "" {
		entry, ok := c.entries.Load(query.NetworkServiceEndpoint.Name)
		if ok {
			values = c.add(entry, values)
		}
		return values
	}

	log.FromContext(ctx).WithField("time", time.Now()).Infof("Range")
	c.entries.Range(func(key string, entry *cacheEntry[registry.NetworkServiceEndpoint]) bool {
		log.FromContext(ctx).WithField("time", time.Now()).Infof("key: %v\n", key)
		log.FromContext(ctx).WithField("time", time.Now()).Infof("entry.value: %v\n", entry.value)
		if subset(query.NetworkServiceEndpoint.NetworkServiceNames, entry.value.NetworkServiceNames) {
			log.FromContext(ctx).WithField("time", time.Now()).Infof("adding entry to nses\n")
			values = c.add(entry, values)
		}
		return true
	})

	log.FromContext(ctx).WithField("time", time.Now()).Infof("values: %v\n", values)

	return values
}
