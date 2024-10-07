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

package nanoid_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/nanoid"
)

const (
	idLen = 30
)

func TestNoCollisions(t *testing.T) {
	used := make(map[string]bool)
	for i := 0; i < 50*1000; i++ {
		id, err := nanoid.GenerateString(idLen)
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
		id, err := nanoid.GenerateString(idLen)
		require.NoError(t, err, "Error generating nanoid")
		for _, char := range id {
			chars[char]++
		}
	}

	require.Equal(t, len(chars), len(nanoid.DefaultAlphabet), "Unexpected number of unique characters")

	maxValue := 0.0
	minValue := math.MaxFloat64
	for _, count := range chars {
		distribution := float64(count*len(nanoid.DefaultAlphabet)) / float64(count*idLen)
		if distribution > maxValue {
			maxValue = distribution
		}
		if distribution < minValue {
			minValue = distribution
		}
	}

	require.True(t, maxValue-minValue <= 0.05, "Distribution is not flat")
}

func TestCustomAlphabet(t *testing.T) {
	alphabet := "abcd"
	id, err := nanoid.GenerateString(idLen, nanoid.WithAlphabet(alphabet))
	require.NoError(t, err)

	for _, c := range id {
		require.Contains(t, alphabet, string(c))
	}
}
