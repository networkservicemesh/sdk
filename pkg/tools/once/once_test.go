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

package once_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/once"
)

func TestOnceUntilSuccess(t *testing.T) {
	var o once.UntilSuccess
	var wg sync.WaitGroup
	var counter int

	operation := func() bool {
		counter++
		return counter == 50
	}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			o.Do(operation)
			wg.Done()
		}()
	}
	wg.Wait()

	require.Equal(t, counter, 50)
}
