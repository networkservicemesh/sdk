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

package connect

import (
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type clientMap struct {
	internalMap sync.Map
}

func newClientMap() clientMap {
	return clientMap{}
}

func (c *clientMap) Delete(key string) {
	c.internalMap.Delete(key)
}

func (c *clientMap) Load(key string) (networkservice.NetworkServiceClient, bool) {
	value, ok := c.internalMap.Load(key)
	var result networkservice.NetworkServiceClient
	if ok {
		result = value.(networkservice.NetworkServiceClient)
	}
	return result, ok
}

func (c *clientMap) LoadOrStore(key string, value networkservice.NetworkServiceClient) (actual networkservice.NetworkServiceClient, loaded bool) {
	rawActual, loaded := c.internalMap.LoadOrStore(key, value)
	return rawActual.(networkservice.NetworkServiceClient), loaded
}

func (c *clientMap) Range(f func(key string, value networkservice.NetworkServiceClient) bool) {
	internalF := func(internalKey, internalValue interface{}) bool {
		s, ok := internalKey.(string)
		if !ok {
			return false
		}
		c, ok := internalValue.(networkservice.NetworkServiceClient)
		if !ok {
			return false
		}
		return f(s, c)
	}
	c.internalMap.Range(internalF)
}

func (c *clientMap) Store(key string, value networkservice.NetworkServiceClient) {
	c.internalMap.Store(key, value)
}
