// Copyright (c) 2022 Doc.ai and/or its affiliates.
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
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/multiexecutor"
)

const (
	totalCount    = 1000
	parallelCount = 10
)

func TestMultiExecutor_AsyncExec(t *testing.T) {
	e := new(multiexecutor.MultiExecutor)

	var data sync.Map
	for i := 0; i < totalCount; i++ {
		id := strconv.Itoa(i % parallelCount)

		k := i
		e.AsyncExec(id, func() {
			val, ok := data.Load(id)
			if ok == false {
				val = 0
			}
			require.Equal(t, k/parallelCount, val.(int))
			data.Store(id, val.(int)+1)
		})
	}

	for i := 0; i < parallelCount; i++ {
		id := strconv.Itoa(i)
		<-e.AsyncExec(id, func() {
			val, ok := data.Load(id)
			require.True(t, ok)
			require.Equal(t, totalCount/parallelCount, val.(int))
		})
	}
}
