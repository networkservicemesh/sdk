// Copyright (c) 2022 Cisco and/or its affiliates.
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

package upstreamrefresh

import (
	"context"

	"github.com/edwarnicke/serialize"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// notifier - notifies all subscribers of an event.
type notifier struct {
	executor serialize.Executor
	channels map[string]chan struct{}
}

func newNotifier() *notifier {
	return &notifier{
		channels: make(map[string]chan struct{}),
	}
}

func (n *notifier) subscribe(id string) {
	if n == nil {
		return
	}
	<-n.executor.AsyncExec(func() {
		n.channels[id] = make(chan struct{})
	})
}

func (n *notifier) get(id string) <-chan struct{} {
	if n == nil {
		return nil
	}
	var ch chan struct{} = nil
	<-n.executor.AsyncExec(func() {
		if v, ok := n.channels[id]; ok {
			ch = v
		}
	})
	return ch
}

func (n *notifier) unsubscribe(id string) {
	if n == nil {
		return
	}
	<-n.executor.AsyncExec(func() {
		if v, ok := n.channels[id]; ok {
			close(v)
		}
		delete(n.channels, id)
	})
}

func (n *notifier) Notify(ctx context.Context, initiatorID string) {
	if n == nil {
		return
	}
	<-n.executor.AsyncExec(func() {
		for k, v := range n.channels {
			if initiatorID == k {
				continue
			}
			log.FromContext(ctx).WithField("upstreamrefresh", "notifier").Debugf("send notification to: %v", k)
			v <- struct{}{}
		}
	})
}

// Notifier - interface for local notifications sending.
type Notifier interface {
	Notify(ctx context.Context, initiatorID string)
}

var _ Notifier = &notifier{}
