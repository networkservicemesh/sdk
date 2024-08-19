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

package dualstackippool

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContains(t *testing.T) {
	ipPool := New()
	ipPool.AddNetString("192.168.0.0/16")
	ipPool.AddNetString("ff80::/64")
	require.True(t, ipPool.ContainsString("192.168.0.1"))
	require.False(t, ipPool.ContainsString("193.169.0.1"))
	require.True(t, ipPool.ContainsString("ff80::ff10"))
	require.False(t, ipPool.ContainsString("ff90::ff10"))
}

//nolint:dupl
func TestPull(t *testing.T) {
	ipPool := New()
	ipPool.AddNetString("192.0.0.0/8")
	require.NotNil(t, ipPool)

	ip, err := ipPool.PullIPString("192.168.0.1/32")
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.168.0.1/32")

	ipPool.AddNetString("ff80::/64")
	ip, err = ipPool.PullIPString("ff80::1/128")
	require.NoError(t, err)
	require.Equal(t, ip.String(), "ff80::1/128")
}

func TestInvalidCIDR(t *testing.T) {
	ipPool := New()
	ipPool.AddNetString("300.168.0.0/16")
	_, err := ipPool.PullIPString("300.168.0.0/32")
	require.Error(t, err)
}
