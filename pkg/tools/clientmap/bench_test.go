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

var ids = []string{
	"00", "01", "02", "03", "04", "05", "06", "07", "08", "09",
	"10", "11", "12", "13", "14", "15", "16", "17", "18", "19",
	"20", "21", "22", "23", "24", "25", "26", "27", "28", "29",
	"30", "31", "32", "33", "34", "35", "36", "37", "38", "39",
	"40", "41", "42", "43", "44", "45", "46", "47", "48", "49",
	"50", "51", "52", "53", "54", "55", "56", "57", "58", "59",
	"60", "61", "62", "63", "64", "65", "66", "67", "68", "69",
	"70", "71", "72", "73", "74", "75", "76", "77", "78", "79",
	"80", "81", "82", "83", "84", "85", "86", "87", "88", "89",
	"90", "91", "92", "93", "94", "95", "96", "97", "98", "99", "100",
}

func BenchmarkMap(b *testing.B) {
	var m clientmap.Map
	b.SetParallelism(1000)
	b.RunParallel(func(pb *testing.PB) {
		client := null.NewClient()
		for i := 0; pb.Next(); i++ {
			id := ids[i%len(ids)]
			switch i % 6 {
			case 0:
				m.Store(id, client)
			case 1:
				client, _ = m.LoadOrStore(id, client)
			case 2:
				client, _ = m.Load(id)
			case 3, 4, 5:
				client, _ = m.LoadAndDelete(id)
			}
		}
	})
}

func BenchmarkRefcountMap(b *testing.B) {
	var m clientmap.RefcountMap
	b.SetParallelism(1000)
	b.RunParallel(func(pb *testing.PB) {
		client := null.NewClient()
		for i := 0; pb.Next(); i++ {
			id := ids[i%len(ids)]
			switch i % 6 {
			case 0:
				m.Store(id, client)
			case 1:
				client, _ = m.LoadOrStore(id, client)
			case 2:
				client, _ = m.Load(id)
			case 3, 4, 5:
				client, _, _ = m.LoadAndDelete(id)
			}
		}
	})
}
