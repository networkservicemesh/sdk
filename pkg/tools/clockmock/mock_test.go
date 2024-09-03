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

package clockmock_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
)

const (
	timeout  = 2 * time.Hour
	testWait = 200 * time.Millisecond
	testTick = testWait / 100
)

func TestMock_SetSpeed(t *testing.T) {
	samples := []struct {
		name                    string
		firstSpeed, secondSpeed float64
	}{
		{
			name:        "From 0",
			firstSpeed:  0,
			secondSpeed: 1,
		},
		{
			name:        "To 0",
			firstSpeed:  1,
			secondSpeed: 0,
		},
		{
			name:        "Same",
			firstSpeed:  1,
			secondSpeed: 1,
		},
		{
			name:        "Increasing to",
			firstSpeed:  0.1,
			secondSpeed: 1,
		},
		{
			name:        "Increasing from",
			firstSpeed:  1,
			secondSpeed: 10,
		},
		{
			name:        "Decreasing to",
			firstSpeed:  10,
			secondSpeed: 1,
		},
		{
			name:        "Decreasing from",
			firstSpeed:  1,
			secondSpeed: 0.1,
		},
	}

	speeds := []struct {
		name       string
		multiplier float64
	}{
		{
			name:       "Slow",
			multiplier: 0.001,
		},
		{
			name:       "Real",
			multiplier: 1,
		},
		{
			name:       "Fast",
			multiplier: 1000,
		},
	}

	for _, sample := range samples {
		//nolint:scopelint
		t.Run(sample.name, func(t *testing.T) {
			for _, speed := range speeds {
				//nolint:scopelint
				t.Run(speed.name, func(t *testing.T) {
					testMockSetSpeed(t, sample.firstSpeed*speed.multiplier, sample.secondSpeed*speed.multiplier)
				})
			}
		})
	}
}

func testMockSetSpeed(t *testing.T, firstSpeed, secondSpeed float64) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const hours = 3

	m := clockmock.New(ctx)

	m.SetSpeed(firstSpeed)

	realStart, mockStart := time.Now(), m.Now()
	for i := 0; i < hours; i++ {
		time.Sleep(testWait)
		m.Add(time.Hour)
	}
	realDuration, mockDuration := time.Since(realStart), m.Since(mockStart)

	m.SetSpeed(secondSpeed)

	realStart, mockStart = time.Now(), m.Now()
	for i := 0; i < hours; i++ {
		time.Sleep(testWait)
		m.Add(time.Hour)
	}
	realDuration += time.Since(realStart)
	mockDuration += m.Since(mockStart)

	mockSpeed := float64(mockDuration-2*hours*time.Hour) / float64(realDuration)
	avgSpeed := (firstSpeed + secondSpeed) / 2

	require.Greater(t, mockSpeed/avgSpeed, 0.6)
	require.Less(t, mockSpeed/avgSpeed, 1.4)
}

func TestMock_Timer(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := clockmock.New(ctx)

	timer := m.Timer(timeout)

	select {
	case <-timer.C():
		require.FailNow(t, "too early")
	case <-time.After(testWait):
	}

	m.Add(timeout / 2)

	select {
	case <-timer.C():
		require.FailNow(t, "too early")
	case <-time.After(testWait):
	}

	m.Add(timeout / 2)

	select {
	case <-timer.C():
	case <-time.After(testWait):
		require.FailNow(t, "too late")
	}
}

func TestMock_Timer_ZeroDuration(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := clockmock.New(ctx)

	timer := m.Timer(0)

	select {
	case <-timer.C():
	case <-time.After(testWait):
		require.FailNow(t, "too late")
	}
}

func TestMock_Timer_Stop(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := clockmock.New(ctx)

	timer := m.Timer(timeout)

	require.True(t, timer.Stop())
	require.False(t, timer.Stop())

	m.Add(timeout)

	select {
	case <-timer.C():
		require.FailNow(t, "is stopped")
	case <-time.After(testWait):
	}
}

func TestMock_Timer_StopResult(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := clockmock.New(ctx)

	timer := m.Timer(timeout)

	m.Add(timeout)
	require.False(t, timer.Stop())

	<-timer.C()
}

func TestMock_Timer_Reset(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := clockmock.New(ctx)

	timer := m.Timer(timeout)

	m.Add(timeout / 2)

	timer.Stop()
	timer.Reset(timeout)

	m.Add(timeout / 2)

	select {
	case <-timer.C():
		require.FailNow(t, "too early")
	case <-time.After(testWait):
	}

	m.Add(timeout / 2)

	select {
	case <-timer.C():
	case <-time.After(testWait):
		require.FailNow(t, "too late")
	}
}

func TestMock_Timer_ResetExpired(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := clockmock.New(ctx)

	timer := m.Timer(timeout)

	m.Add(timeout)

	timer.Stop()
	<-timer.C()
	timer.Reset(timeout)

	m.Add(timeout / 2)

	select {
	case <-timer.C():
		require.FailNow(t, "too early")
	case <-time.After(testWait):
	}

	m.Add(timeout / 2)

	select {
	case <-timer.C():
	case <-time.After(testWait):
		require.FailNow(t, "too late")
	}
}

func TestMock_Timer_Reset_ZeroDuration(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := clockmock.New(ctx)

	timer := m.Timer(timeout)

	timer.Stop()
	timer.Reset(0)

	select {
	case <-timer.C():
	case <-time.After(testWait):
		require.FailNow(t, "too late")
	}
}

func TestMock_AfterFunc(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := clockmock.New(ctx)

	var count int32
	for i := time.Duration(0); i < 10; i++ {
		m.AfterFunc(timeout*i, func() {
			atomic.AddInt32(&count, 1)
		})
	}

	m.Add(4 * timeout)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&count) == 5
	}, testWait, testTick)

	require.Never(t, func() bool {
		return atomic.LoadInt32(&count) > 5
	}, testWait, testTick)

	m.Add(5 * timeout)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&count) == 10
	}, testWait, testTick)
}

func TestMock_AfterFunc_ZeroDuration(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := clockmock.New(ctx)

	var count int32
	m.AfterFunc(0, func() {
		atomic.AddInt32(&count, 1)
	})

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&count) == 1
	}, testWait, testTick)
}

func TestMock_Ticker(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := clockmock.New(ctx)

	ticker := m.Ticker(timeout)

	for i := 0; i < 2; i++ {
		select {
		case <-ticker.C():
			require.FailNow(t, "too early")
		case <-time.After(testWait):
		}

		m.Add(timeout / 2)

		select {
		case <-ticker.C():
			require.FailNow(t, "too early")
		case <-time.After(testWait):
		}

		m.Add(timeout / 2)

		select {
		case <-ticker.C():
		case <-time.After(testWait):
			require.FailNow(t, "too late")
		}
	}
}

func TestMock_WithDeadline(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := clockmock.New(ctx)

	deadlineCtx, deadlineCtxCancel := m.WithDeadline(context.Background(), m.Now().Add(timeout))
	defer deadlineCtxCancel()

	select {
	case <-deadlineCtx.Done():
		require.FailNow(t, "too early")
	case <-time.After(testWait):
		require.NoError(t, deadlineCtx.Err())
	}

	m.Add(timeout)

	select {
	case <-deadlineCtx.Done():
		require.Error(t, deadlineCtx.Err())
	case <-time.After(testWait):
		require.FailNow(t, "too late")
	}
}

func TestMock_WithDeadline_Expired(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := clockmock.New(ctx)

	deadlineCtx, deadlineCtxCancel := m.WithDeadline(context.Background(), m.Now())
	defer deadlineCtxCancel()

	select {
	case <-deadlineCtx.Done():
		require.Error(t, deadlineCtx.Err())
	case <-time.After(testWait):
		require.FailNow(t, "too late")
	}
}

func TestMock_WithDeadline_ParentCanceled(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := clockmock.New(ctx)

	parentCtx, parentCancel := context.WithCancel(context.Background())

	deadlineCtx, deadlineCtxCancel := m.WithDeadline(parentCtx, m.Now().Add(timeout))
	defer deadlineCtxCancel()

	select {
	case <-deadlineCtx.Done():
		require.FailNow(t, "too early")
	case <-time.After(testWait):
		require.NoError(t, deadlineCtx.Err())
	}

	parentCancel()

	select {
	case <-deadlineCtx.Done():
		require.Error(t, deadlineCtx.Err())
	case <-time.After(testWait):
		require.FailNow(t, "too late")
	}
}

func TestMock_WithTimeout(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := clockmock.New(ctx)

	timeoutCtx, timeoutCtxCancel := m.WithTimeout(context.Background(), timeout)
	defer timeoutCtxCancel()

	select {
	case <-timeoutCtx.Done():
		require.FailNow(t, "too early")
	case <-time.After(testWait):
		require.NoError(t, timeoutCtx.Err())
	}

	m.Add(timeout)

	select {
	case <-timeoutCtx.Done():
		require.Error(t, timeoutCtx.Err())
	case <-time.After(testWait):
		require.FailNow(t, "too late")
	}
}
