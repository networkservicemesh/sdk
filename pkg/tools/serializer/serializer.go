// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package serializer provides true FIFO by-ID lock
package serializer

import (
	"sync"

	"github.com/edwarnicke/serialize"
)

var dummyChan = dummyChannel()

// Serializer is a true FIFO by-ID lock:
// 1. A -> lock 'a' -- A locked 'a'
// 2. B -> lock 'a' -- B waiting 'a'
// 3. C -> lock 'b' -- C locked 'b'
// 4. A -> unlock 'a', D -> lock 'a' -- B locked 'a', D waiting 'a'
type Serializer struct {
	init     sync.Once
	executor serialize.Executor
	chans    map[string][]chan struct{}
}

// Lock tries to acquire by-ID lock:
// * if lock is free it returns unlock function
// * if lock is not free it waits until it becomes free and returns unlock function
func (s *Serializer) Lock(id string) func() {
	s.init.Do(func() {
		s.chans = map[string][]chan struct{}{}
	})

	var waitChan, updateChan chan struct{}
	<-s.executor.AsyncExec(func() {
		chans := s.chans[id]
		if size := len(chans); size != 0 {
			waitChan = chans[size-1]
		} else {
			waitChan = dummyChan
		}

		updateChan = make(chan struct{})
		s.chans[id] = append(s.chans[id], updateChan)
	})

	<-waitChan

	return func() {
		close(updateChan)
		s.executor.AsyncExec(func() {
			s.chans[id] = s.chans[id][1:]
			if len(s.chans[id]) == 0 {
				delete(s.chans, id)
			}
		})
	}
}
