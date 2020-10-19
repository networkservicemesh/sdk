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
	lock := fifosync.Mutex{}
	lock.Lock()

	wg := sync.WaitGroup{}
	wg.Add(parallelCount * 2)

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
		// workers should be locked in the same order they appear
		<-time.After(time.Millisecond)
	}

	lock.Unlock()

	// we need to be sure that new workers would not break existing FIFO order
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
