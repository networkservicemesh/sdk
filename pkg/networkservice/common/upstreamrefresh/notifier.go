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

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// notifier - notifies all subscribers of an event
type notifier struct {
	channels notifierMap
}

func newNotifier() *notifier {
	return &notifier{}
}

func (n *notifier) subscribe(id string) {
	if n == nil {
		return
	}
	n.unsubscribe(id)
	n.channels.Store(id, make(chan struct{}))
}

func (n *notifier) get(id string) <-chan struct{} {
	if n == nil {
		return nil
	}
	if v, ok := n.channels.Load(id); ok {
		return v
	}
	return nil
}

func (n *notifier) unsubscribe(id string) {
	if n == nil {
		return
	}
	if v, ok := n.channels.LoadAndDelete(id); ok {
		close(v)
	}
}

func (n *notifier) notify(ctx context.Context, initiatorID string) {
	if n == nil {
		return
	}
	n.channels.Range(func(key string, value typeCh) bool {
		if initiatorID == key {
			return true
		}
		log.FromContext(ctx).WithField("upstreamrefresh", "notifier").Debug("send notification to: %v", key)
		value <- struct{}{}
		return true
	})
}
