// Copyright (c) 2024 Cisco and/or its affiliates.
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

package kernel_test

import (
	"math"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/stretchr/testify/require"
)

const (
	idLen = 30
)

func TestNoCollisions(t *testing.T) {
	used := make(map[string]bool)
	for i := 0; i < 50*1000; i++ {
		id, err := kernel.GenerateRandomString(idLen)
		require.NoError(t, err)
		_, ok := used[id]
		require.False(t, ok, "Collision detected for id: %s", id)
		used[id] = true
	}
}

func TestFlatDistribution(t *testing.T) {
	count := 100 * 1000

	chars := make(map[rune]int)
	for i := 0; i < count; i++ {
		id, err := kernel.GenerateRandomString(idLen)
		require.NoError(t, err, "Error generating nanoid")
		for _, char := range id {
			chars[char]++
		}
	}

	require.Equal(t, len(chars), len(kernel.Alphabet), "Unexpected number of unique characters")

	max := 0.0
	min := math.MaxFloat64
	for _, count := range chars {
		distribution := float64(count*len(kernel.Alphabet)) / float64(count*idLen)
		if distribution > max {
			max = distribution
		}
		if distribution < min {
			min = distribution
		}
	}

	require.True(t, max-min <= 0.05, "Distribution is not flat")
}
