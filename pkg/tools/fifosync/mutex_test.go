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

package fifosync_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/tools/fifosync"
)

func TestMutex(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(2 * parallelCount)

	lock := fifosync.Mutex{}
	lock.Lock()

	var count int
	for i := 0; i < parallelCount; i++ {
		id := i
		go func() {
			lock.Lock()
			defer lock.Unlock()

			assert.Equal(t, id, count)
			count++

			wg.Done()
		}()
		// lock workers in the same order they appear
		<-time.After(time.Millisecond)
	}

	lock.Unlock()

	// be sure that new workers would not break existing FIFO order
	for i := 0; i < parallelCount; i++ {
		go func() {
			lock.Lock()
			defer lock.Unlock()

			count++

			wg.Done()
		}()
	}

	wg.Wait()
}

func TestMutate(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(3 * parallelCount)

	barrier := sync.WaitGroup{}
	barrier.Add(1)

	locks := make([]fifosync.Mutex, parallelCount)
	nextLocks := make([]fifosync.Mutex, parallelCount)

	// lock nextLocks and be sure that everybody are blocked on them
	for i := 0; i < parallelCount; i++ {
		id := i
		go func() {
			nextLocks[id].Lock()
			defer nextLocks[id].Unlock()

			wg.Done()
			barrier.Wait()

			time.Sleep(time.Millisecond)

			wg.Done()
		}()
	}

	counts := make([]int, parallelCount)
	for i := 0; i < parallelCount; i++ {
		id := i
		go func() {
			locks[id].Lock()

			wg.Done()
			barrier.Wait()

			fifosync.Mutate(&locks[id], &nextLocks[id])
			defer nextLocks[id].Unlock()

			assert.Equal(t, 0, counts[id])
			counts[id]++

			wg.Done()
		}()
	}

	// try to lock both mutexes before the mutate happens
	for i := 0; i < parallelCount; i++ {
		id := i
		go func() {
			wg.Done()

			locks[id].Lock()
			nextLocks[id].Lock()

			counts[id]++

			locks[id].Unlock()
			nextLocks[id].Unlock()

			wg.Done()
		}()
	}

	wg.Wait()
	wg.Add(3 * parallelCount)
	barrier.Done()

	wg.Wait()
}
