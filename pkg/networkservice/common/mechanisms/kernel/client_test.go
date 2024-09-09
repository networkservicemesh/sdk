// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
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
	"net/url"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/tools/nanoid"
)

var netNSURL = (&url.URL{Scheme: "file", Path: "/proc/thread-self/ns/net"}).String()

func TestKernelMechanismClient_ShouldSetInterfaceName(t *testing.T) {
	var expectedIfaceName string
	for i := 0; i < kernelmech.LinuxIfMaxLength; i++ {
		expectedIfaceName += "a"
	}

	c := kernel.NewClient(kernel.WithInterfaceName(expectedIfaceName + "long-suffix"))

	req := &networkservice.NetworkServiceRequest{}
	_, err := c.Request(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, req.GetMechanismPreferences(), 1)
	require.Equal(t, expectedIfaceName, req.GetMechanismPreferences()[0].GetParameters()[kernelmech.InterfaceNameKey])
}

func TestKernelMechanismClient_ShouldNotDoublingMechanisms(t *testing.T) {
	c := kernel.NewClient()

	req := &networkservice.NetworkServiceRequest{}

	for i := 0; i < 10; i++ {
		_, err := c.Request(context.Background(), req)
		require.NoError(t, err)
		require.Len(t, req.GetMechanismPreferences(), 1)
	}
}

func TestKernelMechanismClient_ShouldSetValidNetNSURL(t *testing.T) {
	c := kernel.NewClient()

	req := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			kernelmech.New("invalid-url"),
		},
	}

	_, err := c.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, netNSURL, req.GetMechanismPreferences()[0].GetParameters()[kernelmech.NetNSURL])
}

func TestKernelMechanismClient_ShouldSetRandomInteraceName(t *testing.T) {
	c := kernel.NewClient()
	req := &networkservice.NetworkServiceRequest{Connection: &networkservice.Connection{NetworkService: "nsm"}}

	_, err := c.Request(context.Background(), req)
	require.NoError(t, err)

	ifname := req.GetMechanismPreferences()[0].GetParameters()[kernelmech.InterfaceNameKey]

	require.Len(t, ifname, kernelmech.LinuxIfMaxLength)
	require.True(t, strings.HasPrefix(ifname, "nsm"))
	for i := 0; i < kernelmech.LinuxIfMaxLength; i++ {
		require.Contains(t, nanoid.DefaultAlphabet+"-", string(ifname[i]))
	}
}

func TestKernelMechanismClient_FailedToGenerateRandomName(t *testing.T) {
	c := kernel.NewClient(kernel.WithInterfaceNameGenerator(func(_ string) (string, error) {
		return "", errors.New("failed to generate bytes")
	}))
	req := &networkservice.NetworkServiceRequest{}

	_, err := c.Request(context.Background(), req)
	require.Error(t, err)
}
