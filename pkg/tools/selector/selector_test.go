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

// Package selector provides a selection by any algorithm any item
package selector

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/algorithm/roundrobin"
)

//
type stringArray struct {
	items []string
}

func (a stringArray) GetOptions() ([]interface{}, error) {
	result := make([]interface{}, 0)
	for _, item := range a.items {
		result = append(result, item)
	}
	return result, nil
}

func newStringArray(strs []string) OptionsProvider {
	return &stringArray{items: strs}
}

func TestSelector(t *testing.T) {
	str := []string{"1", "2"}
	selector := New(&roundrobin.IndexedDecider{}, newStringArray(str))
	tmp, _ := selector.Select()
	require.Equal(t, "2", tmp)
	tmp, _ = selector.Select()
	require.Equal(t, "1", tmp)
	tmp, _ = selector.Select()
	require.Equal(t, "2", tmp)
}
