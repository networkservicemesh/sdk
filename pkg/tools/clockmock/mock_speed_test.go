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
	"testing"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMock_SetSpeed(t *testing.T) {
	t.Parallel()
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
		sample := sample
		// nolint:scopelint
		t.Run(sample.name, func(t *testing.T) {
			for _, speed := range speeds {
				speed := speed
				// nolint:scopelint
				t.Run(speed.name, func(t *testing.T) {
					t.Parallel()
					testMockSetSpeed(t, sample.firstSpeed*speed.multiplier, sample.secondSpeed*speed.multiplier)
				})
			}
		})
	}
}

func testMockSetSpeed(t *testing.T, firstSpeed, secondSpeed float64) {
	t.Cleanup(func() { goleak.VerifyNone(t, goleak.IgnoreCurrent()) })

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
