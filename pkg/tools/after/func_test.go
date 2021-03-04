// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package after_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/tools/after"
)

const (
	testDuration = 500 * time.Millisecond
	timeTick     = testDuration / 10
)

func TestFunc_ExecutesOnTime(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	intPtr := new(int32)

	after.NewFunc(context.Background(), time.Now().Add(testDuration), func() {
		atomic.StoreInt32(intPtr, 1)
	})

	// Wait for some time
	<-time.After(8 * timeTick)

	require.Equal(t, int32(0), atomic.LoadInt32(intPtr))

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(intPtr) == 1
	}, 4*timeTick, timeTick)
}

func TestFunc_StopsWithContext(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())

	intPtr := new(int32)

	after.NewFunc(ctx, time.Now().Add(testDuration), func() {
		atomic.StoreInt32(intPtr, 1)
	})

	cancel()

	require.Never(t, func() bool {
		return atomic.LoadInt32(intPtr) == 1
	}, 12*timeTick, timeTick)
}

func TestFunc_Stop(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ints := make([]int32, 100)

	var wg sync.WaitGroup
	for i := range ints {
		intPtr := &ints[i]

		f := after.NewFunc(context.Background(), time.Now().Add(testDuration), func() {
			atomic.StoreInt32(intPtr, 1)
		})

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			time.Sleep(testDuration/time.Duration(i+1) + 2*timeTick)

			if f.Stop() {
				assert.Equal(t, int32(0), *intPtr)
			} else {
				assert.Equal(t, int32(1), *intPtr)
			}
		}(i)
	}
	wg.Wait()
}

func TestFunc_Resume(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ints := make([]int32, 100)

	var wg sync.WaitGroup
	for i := range ints {
		intPtr := &ints[i]

		expireTime := time.Now().Add(testDuration)
		f := after.NewFunc(context.Background(), expireTime, func() {
			atomic.StoreInt32(intPtr, 1)
		})

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			time.Sleep(testDuration/time.Duration(i+1) + 2*timeTick)

			if !f.Stop() {
				return
			}
			f.Resume()

			assert.Eventually(t, func() bool {
				return atomic.LoadInt32(intPtr) == 1
			}, time.Until(expireTime)+2*timeTick, timeTick)
		}(i)
	}
	wg.Wait()
}

func TestFunc_Reset(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ints := make([]int32, 100)

	var wg sync.WaitGroup
	for i := range ints {
		intPtr := &ints[i]

		f := after.NewFunc(context.Background(), time.Now().Add(testDuration), func() {
			atomic.StoreInt32(intPtr, 1)
		})

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			time.Sleep(testDuration/time.Duration(i+1) + timeTick)

			f.Reset(time.Now().Add(testDuration))

			atomic.CompareAndSwapInt32(intPtr, 1, 0)

			// Wait for some time
			<-time.After(8 * timeTick)

			assert.Equal(t, int32(0), atomic.LoadInt32(intPtr))

			assert.Eventually(t, func() bool {
				return atomic.LoadInt32(intPtr) == 1
			}, 4*timeTick, timeTick)
		}(i)
	}
	wg.Wait()
}
