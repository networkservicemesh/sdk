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

package beginrecursive1_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/edwarnicke/serialize"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/stretchr/testify/require"
)

func executeTwoTimes(ctx context.Context, thread int, buffer []int, executor *serialize.Executor, wg *sync.WaitGroup) {

	if thread != 0 {
		<-executor.AsyncExec(func() {
			log.FromContext(ctx).Infof("Thread [%v] started a short exec", thread)
			time.Sleep(time.Millisecond * 2)
			log.FromContext(ctx).Infof("Thread [%v] finished a short exec", thread)
		})
		log.FromContext(ctx).Infof("Thread [%v] exited a short exec", thread)
	}

	<-executor.AsyncExec(func() {
		log.FromContext(ctx).Infof("Thread [%v] started a long exec", thread)
		time.Sleep(time.Millisecond * 50)
		buffer = append(buffer, thread)
		log.FromContext(ctx).Infof("Thread [%v] finished a long exec", thread)
	})

	wg.Done()
}

func TestExecutor(t *testing.T) {
	count := 3

	ctx := context.Background()

	var executor serialize.Executor
	var wg sync.WaitGroup
	buffer := make([]int, 0)
	wg.Add(count)

	for i := 0; i < count; i++ {
		local := i
		go executeTwoTimes(ctx, local, buffer, &executor, &wg)
		time.Sleep(time.Millisecond * 20)
	}

	wg.Wait()
	for i, val := range buffer {
		require.Equal(t, i, val)
	}
}
