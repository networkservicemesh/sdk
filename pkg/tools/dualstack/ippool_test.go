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

package dualstack

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContains(t *testing.T) {
	ipPool := New()
	ipPool.AddIPNetString("192.168.0.0/16")
	ipPool.AddIPNetString("ff80::/64")
	require.True(t, ipPool.ContainsIPString("192.168.0.1/32"))
	require.False(t, ipPool.ContainsIPString("193.169.0.1/32"))
	require.True(t, ipPool.ContainsIPString("ff80::ff10/128"))
	require.False(t, ipPool.ContainsIPString("ff90::ff10/128"))
}

//nolint:dupl
func TestPull(t *testing.T) {
	ipPool := New()
	ipPool.AddIPNetString("192.0.0.0/8")
	require.NotNil(t, ipPool)

	ip, err := ipPool.PullIPString("192.168.0.1/32")
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.168.0.1/32")

	ipPool.AddIPNetString("ff80::/64")
	ip, err = ipPool.PullIPString("ff80::1/128")
	require.NoError(t, err)
	require.Equal(t, ip.String(), "ff80::1/128")
}

func TestInvalidCIDR(t *testing.T) {
	ipPool := New()
	ipPool.AddIPNetString("300.168.0.0/16")
	_, err := ipPool.PullIPString("300.168.0.0/32")
	require.Error(t, err)
}

func TestDualStack(t *testing.T) {
	ipPool := New()
	ipPool.AddIPNetString("172.16.1.100/27")
	ipPool.AddIPNetString("2001:db8::/116")
	require.True(t, ipPool.ContainsIPNetString("172.16.1.97/32"))
}
