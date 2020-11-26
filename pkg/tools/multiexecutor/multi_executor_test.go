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

package multiexecutor_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/tools/multiexecutor"
)

const (
	parallelCount = 1000
)

var ids = []string{
	"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
}

func TestExecutor_AsyncExec(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	var wg sync.WaitGroup
	wg.Add(parallelCount)

	var e multiexecutor.Executor
	var vals intMap
	for i := 0; i < parallelCount; i++ {
		k := i
		id := ids[i%len(ids)]
		e.AsyncExec(id, func() {
			defer wg.Done()
			val := vals.load(id)
			assert.Equal(t, k/len(ids), val)
			vals.store(id, val+1)
		})
	}

	wg.Wait()
}

type intMap sync.Map

func (m *intMap) load(key string) int {
	val, ok := (*sync.Map)(m).Load(key)
	if !ok {
		return 0
	}
	return val.(int)
}

func (m *intMap) store(key string, val int) {
	(*sync.Map)(m).Store(key, val)
}
