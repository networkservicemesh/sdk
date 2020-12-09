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

package clientmap_test

import (
	"testing"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/tools/clientmap"
)

const (
	parallelCount = 20
)

var ids = []string{
	"00", "01", "02", "03", "04", "05", "06", "07", "08", "09", "10",
}

func BenchmarkMap(b *testing.B) {
	var m clientmap.Map
	client := null.NewClient()
	b.SetParallelism(parallelCount)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			id := ids[i%len(ids)]
			switch i % 6 {
			case 0:
				m.Store(id, client)
			case 1:
				_, _ = m.LoadOrStore(id, client)
			case 2:
				_, _ = m.Load(id)
			case 3, 4, 5:
				_, _ = m.LoadAndDelete(id)
			}
		}
	})
}

func BenchmarkRefcountMap(b *testing.B) {
	var m clientmap.RefcountMap
	client := null.NewClient()
	b.SetParallelism(parallelCount)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			id := ids[i%len(ids)]
			switch i % 6 {
			case 0:
				m.Store(id, client)
			case 1:
				_, _ = m.LoadOrStore(id, client)
			case 2:
				_, _ = m.Load(id)
			case 3, 4, 5:
				_, _, _ = m.LoadAndDelete(id)
			}
		}
	})
}
