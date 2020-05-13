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

package repository_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/repository"
)

func TestRepository_PutAndGet(t *testing.T) {
	r := repository.NewGenericMemoryRepository(func(generic *repository.Generic) string {
		return fmt.Sprint(generic)
	})
	var wg sync.WaitGroup
	wg.Add(2)
	g1 := new(repository.Generic)
	go func() {
		defer wg.Done()
		r.Put(g1)
	}()
	g2 := new(repository.Generic)
	go func() {
		defer wg.Done()
		r.Put(g2)
	}()
	wg.Wait()
	require.Equal(t, g1, r.Get(fmt.Sprint(g1)))
	require.Equal(t, g2, r.Get(fmt.Sprint(g2)))
}

func TestRepository_GetAllByFilter(t *testing.T) {
	r := repository.NewGenericMemoryRepository(func(generic *repository.Generic) string {
		return fmt.Sprint(generic)
	})
	const count = 10
	var wg sync.WaitGroup
	wg.Add(count)
	var items []*repository.Generic
	for i := 0; i < count; i++ {
		item := new(repository.Generic)
		items = append(items, item)
		go func() {
			defer wg.Done()
			r.Put(item)
		}()
	}
	wg.Wait()
	firstAndLast := r.GetAllByFilter(func(item *repository.Generic) bool {
		return item == items[0] || item == items[len(items)-1]
	})
	require.EqualValues(t, []*repository.Generic{items[0], items[len(items)-1]}, firstAndLast)
}
