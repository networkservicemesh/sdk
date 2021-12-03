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

// Package notifcator creates Notificator object which is simply implements publisher/subscriber pattern
package notifcator

import (
	"context"
	"sync"
	"time"
)

type key struct{}

// Notificator object for pub/sub mechanism
type Notificator struct {
	pub  *time.Ticker
	subs []chan struct{}
	lock sync.Mutex
	done chan struct{}
}

// WithNotificator checks if context already contains Notificator and if it's not then creates(and start immediately) new Notificator
func WithNotificator(ctx context.Context) context.Context {
	val := FromContext(ctx)
	if val != nil {
		return ctx
	}

	n := &Notificator{
		pub:  time.NewTicker(time.Minute),
		subs: []chan struct{}{},
	}

	ctx = context.WithValue(ctx, key{}, n)

	go func() {
		for {
			select {
			case <-n.pub.C:
				n.notify()
			case <-ctx.Done():
				n.pub.Stop()
				n.close()
			}
		}
	}()

	return ctx
}

// FromContext returns Notificator from context if it is already present
func FromContext(ctx context.Context) *Notificator {
	val := ctx.Value(key{})
	if val == nil {
		return nil
	}

	rv, ok := val.(*Notificator)
	if !ok || rv == nil {
		return nil
	}

	return rv
}

// AddSubscribers adds new channels to list of subscribers
func (n *Notificator) AddSubscribers(newSubs ...chan struct{}) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.subs = append(n.subs, newSubs...)
}

// Done returns done channel
func (n *Notificator) Done() chan struct{} {
	return n.done
}

func (n *Notificator) close() {
	n.lock.Lock()
	defer n.lock.Unlock()

	for range n.subs {
		n.done <- struct{}{}
	}
}

func (n *Notificator) notify() {
	n.lock.Lock()
	defer n.lock.Unlock()

	for _, s := range n.subs {
		s <- struct{}{}
	}
}
