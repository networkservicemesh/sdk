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

package kernel_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/tools/nanoid"
)

func TestKernelMechanismServer_ShouldSetInterfaceName(t *testing.T) {
	var expectedIfaceName string
	for i := 0; i < kernelmech.LinuxIfMaxLength; i++ {
		expectedIfaceName += "a"
	}

	s := kernel.NewServer(kernel.WithInterfaceName(expectedIfaceName + "long-suffix"))

	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Mechanism: kernelmech.New(""),
		},
	}
	conn, err := s.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, expectedIfaceName, conn.GetMechanism().GetParameters()[kernelmech.InterfaceNameKey])

	// Refresh
	req.Connection = conn.Clone()
	conn, err = s.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, expectedIfaceName, conn.GetMechanism().GetParameters()[kernelmech.InterfaceNameKey])
}

func TestKernelMechanismServer_ShouldSetValidNetNSURL(t *testing.T) {
	s := kernel.NewServer()

	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Mechanism: kernelmech.New("invalid-url"),
		},
	}

	conn, err := s.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, netNSURL, conn.GetMechanism().GetParameters()[kernelmech.NetNSURL])

	// Refresh
	req.Connection = conn.Clone()
	conn, err = s.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, netNSURL, conn.GetMechanism().GetParameters()[kernelmech.NetNSURL])
}

func TestKernelMechanismServer_ShouldSetRandomInteraceName(t *testing.T) {
	s := kernel.NewServer()
	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Mechanism:      kernelmech.New(""),
			NetworkService: "nsm-dfs422343tsdf543",
		},
	}

	conn, err := s.Request(context.Background(), req)
	require.NoError(t, err)
	ifname := conn.GetMechanism().GetParameters()[kernelmech.InterfaceNameKey]

	require.Len(t, ifname, kernelmech.LinuxIfMaxLength)
	require.True(t, strings.HasPrefix(ifname, "nsm"))
	for i := 0; i < kernelmech.LinuxIfMaxLength; i++ {
		require.Contains(t, nanoid.DefaultAlphabet+"-", string(ifname[i]))
	}

	// Refresh
	req.Connection = conn.Clone()
	conn, err = s.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, ifname, conn.GetMechanism().GetParameters()[kernelmech.InterfaceNameKey])
}

func TestKernelMechanismServer_FailedToGenerateRandomName(t *testing.T) {
	s := kernel.NewServer(kernel.WithInterfaceNameGenerator(func(_ string) (string, error) {
		return "", errors.New("failed to generate bytes")
	}))
	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Mechanism: kernelmech.New(""),
		},
	}

	_, err := s.Request(context.Background(), req)
	require.Error(t, err)
}
