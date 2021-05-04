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
	testWait = 100 * time.Millisecond
	testTick = testWait / 100
)

func TestMock_Start(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	m := clockmock.New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const speed = 1000.0
	const hours = 7

	m.Start(ctx, speed)

	realStart, mockStart := time.Now(), m.Now()
	for i := 0; i < hours; i++ {
		time.Sleep(testWait)
		m.Add(time.Hour)
	}
	realDuration, mockDuration := time.Since(realStart), m.Since(mockStart)

	mockSpeed := float64((mockDuration - hours*time.Hour) / realDuration)

	require.Greater(t, mockSpeed/speed, 0.9)
	require.Less(t, mockSpeed/speed, 1.1)
}

func TestMock_Timer(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	m := clockmock.New()

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

	m := clockmock.New()

	timer := m.Timer(0)

	select {
	case <-timer.C():
	case <-time.After(testWait):
		require.FailNow(t, "too late")
	}
}

func TestMock_Timer_Stop(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	m := clockmock.New()

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

	m := clockmock.New()

	timer := m.Timer(timeout)

	m.Add(timeout)
	require.False(t, timer.Stop())

	<-timer.C()
}

func TestMock_Timer_Reset(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	m := clockmock.New()

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

	m := clockmock.New()

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

	m := clockmock.New()

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

	m := clockmock.New()

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

	m := clockmock.New()

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

	m := clockmock.New()

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

	m := clockmock.New()

	ctx, cancel := m.WithDeadline(context.Background(), m.Now().Add(timeout))
	defer cancel()

	select {
	case <-ctx.Done():
		require.FailNow(t, "too early")
	case <-time.After(testWait):
		require.NoError(t, ctx.Err())
	}

	m.Add(timeout)

	select {
	case <-ctx.Done():
		require.Error(t, ctx.Err())
	case <-time.After(testWait):
		require.FailNow(t, "too late")
	}
}

func TestMock_WithDeadline_Expired(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	m := clockmock.New()

	ctx, cancel := m.WithDeadline(context.Background(), m.Now())
	defer cancel()

	select {
	case <-ctx.Done():
		require.Error(t, ctx.Err())
	case <-time.After(testWait):
		require.FailNow(t, "too late")
	}
}

func TestMock_WithDeadline_ParentCanceled(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	m := clockmock.New()

	parentCtx, parentCancel := context.WithCancel(context.Background())

	ctx, cancel := m.WithDeadline(parentCtx, m.Now().Add(timeout))
	defer cancel()

	select {
	case <-ctx.Done():
		require.FailNow(t, "too early")
	case <-time.After(testWait):
		require.NoError(t, ctx.Err())
	}

	parentCancel()

	select {
	case <-ctx.Done():
		require.Error(t, ctx.Err())
	case <-time.After(testWait):
		require.FailNow(t, "too late")
	}
}

func TestMock_WithTimeout(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	m := clockmock.New()

	ctx, cancel := m.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		require.FailNow(t, "too early")
	case <-time.After(testWait):
		require.NoError(t, ctx.Err())
	}

	m.Add(timeout)

	select {
	case <-ctx.Done():
		require.Error(t, ctx.Err())
	case <-time.After(testWait):
		require.FailNow(t, "too late")
	}
}
