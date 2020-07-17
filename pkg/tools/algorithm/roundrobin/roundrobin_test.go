// Copyright (c) 2019-2020 VMware, Inc.
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

package roundrobin

import (
	"sync/atomic"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var tests = []struct {
	size int
}{
	{
		size: 0,
	},
	{
		size: 10,
	},
	{
		size: 20,
	},
	{
		size: 50,
	},
	{
		size: 100,
	},
}

func Test_RoundRobin_Index(t *testing.T) {
	for i := range tests {
		test := &tests[i]
		logrus.Infof("simple (%v)", test.size)

		rr := &RoundRobin{}
		for i := 0; i < 10; i++ {
			for k := 0; k < test.size; k++ {
				assert.Equal(t, k, rr.Index(test.size))
			}
		}
	}
}

var testsConcurrent = []struct {
	size  int
	limit int
}{
	{
		size:  10,
		limit: 1,
	},
	{
		size:  10,
		limit: 5,
	},
	{
		size:  10,
		limit: 10,
	},
	{
		size:  100,
		limit: 50,
	},
	{
		size:  100,
		limit: 100,
	},
}

func Test_RoundRobin_IndexConcurrent(t *testing.T) {
	for i := range testsConcurrent {
		test := &testsConcurrent[i]
		logrus.Infof("concurrent (%v of %v)", test.limit, test.size)

		rr := &RoundRobin{}
		flags := make([]bool, test.size)
		limit := int32(test.limit)
		ready := make(chan bool, 10)
		for k := 0; k < 10; k++ {
			go flagPicker(t, rr, flags, &limit, ready)
		}
		for k := 0; k < 10; k++ {
			<-ready
		}

		for k, flag := range flags {
			if k < test.limit {
				assert.True(t, flag, "haven't selected index = %v", k)
			} else {
				assert.False(t, flag, "selected index = %v", k)
			}
		}
	}
}

func flagPicker(t *testing.T, rr *RoundRobin, flags []bool, limit *int32, ready chan<- bool) {
	for atomic.AddInt32(limit, -1) >= 0 {
		idx := rr.Index(len(flags))
		assert.False(t, flags[idx], "double selected index = %v", idx)
		flags[idx] = true
	}
	ready <- true
}
