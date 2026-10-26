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

package vl3dns

import (
	"github.com/edwarnicke/serialize"
)

// notifier - notifies all subscribers of an event
type vl3DNSNotifier struct {
	executor serialize.Executor
	channels map[string]chan struct{}
}

func newNotifier() *vl3DNSNotifier {
	return &vl3DNSNotifier{
		channels: make(map[string]chan struct{}),
	}
}

func (n *vl3DNSNotifier) subscribe(id string) <-chan struct{} {
	if n == nil {
		return nil
	}
	var r chan struct{}
	<-n.executor.AsyncExec(func() {
		n.channels[id] = make(chan struct{})
		r = n.channels[id]
	})
	return r
}

func (n *vl3DNSNotifier) unsubscribe(id string) {
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

func (n *vl3DNSNotifier) notify() {
	if n == nil {
		return
	}
	<-n.executor.AsyncExec(func() {
		for _, v := range n.channels {
			v <- struct{}{}
		}
	})
}
