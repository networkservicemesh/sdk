// Copyright (c) 2022 Cisco and/or its affiliates.
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

package cidr_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/cidr"
)

func Test_CidrGroupsDecoder_EmptyInput(t *testing.T) {
	var expected [][]*net.IPNet
	var groups cidr.Groups
	err := groups.Decode(``)
	require.NoError(t, err)
	require.Equal(t, expected, [][]*net.IPNet(groups))
}

func Test_CidrGroupsDecoder_CorrectInput(t *testing.T) {
	_, cidr1, err := net.ParseCIDR("172.168.1.0/24")
	require.NoError(t, err)
	_, cidr2, err := net.ParseCIDR("172.168.2.0/24")
	require.NoError(t, err)
	_, cidr3, err := net.ParseCIDR("172.168.3.0/24")
	require.NoError(t, err)

	expected := [][]*net.IPNet{
		{cidr1},
		{cidr1, cidr2},
		{cidr1},
		{cidr1, cidr2, cidr3},
	}
	var groups cidr.Groups
	err = groups.Decode(fmt.Sprintf("[%v], [%v, %v], %v, [%v, %v , %v]",
		cidr1.String(),
		cidr1.String(), cidr2.String(),
		cidr1.String(),
		cidr1.String(), cidr2.String(), cidr3.String()))
	require.NoError(t, err)
	require.Equal(t, expected, [][]*net.IPNet(groups))
}

func Test_CidrGroupsDecoder_WrongInput(t *testing.T) {
	cidr1 := "172.168.1.0/24"
	var groups cidr.Groups
	err := groups.Decode(fmt.Sprintf("[%[1]v, %[1]v], [[%[1]v],,[%[1]v]", cidr1))
	require.Error(t, err)

	err = groups.Decode(fmt.Sprintf("[%[1]v, %[1]v", cidr1))
	require.Error(t, err)

	err = groups.Decode(fmt.Sprintf("[%[1]v, %[1]v],", cidr1))
	require.Error(t, err)

	err = groups.Decode(fmt.Sprintf("[%[1]v, %[1]v][%[1]v]", cidr1))
	require.Error(t, err)

	err = groups.Decode(fmt.Sprintf("[%[1]v, %[1]v],[]", cidr1))
	require.Error(t, err)

	err = groups.Decode(fmt.Sprintf("[%[1]v,, %[1]v]", cidr1))
	require.Error(t, err)

	err = groups.Decode(fmt.Sprintf("[%[1]v, %[1]v]", "172.168.1./24"))
	require.Error(t, err)
}
