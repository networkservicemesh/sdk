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

package vl3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/ipcontext/vl3"
)

func TestSubscribtions(t *testing.T) {
	counter := 0
	ipam := new(vl3.IPAM)
	unsub1 := ipam.Subscribe(func() {
		counter++
	})
	unsub2 := ipam.Subscribe(func() {
		counter += 2
	})

	err := ipam.Reset("10.0.0.1/24")
	require.NoError(t, err)
	require.Equal(t, counter, 3)

	unsub2()
	err = ipam.Reset("10.0.0.1/24")
	require.NoError(t, err)
	require.Equal(t, counter, 4)
	unsub1()

	err = ipam.Reset("10.0.0.1/24")
	require.NoError(t, err)
	require.Equal(t, counter, 4)
}
