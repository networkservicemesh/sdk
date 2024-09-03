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

package mechanismpriority_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/srv6"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/wireguard"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismpriority"
)

func TestMechanismPriorityClient_Request(t *testing.T) {
	request := func() *networkservice.NetworkServiceRequest {
		return &networkservice.NetworkServiceRequest{
			MechanismPreferences: []*networkservice.Mechanism{
				{
					Cls:  cls.REMOTE,
					Type: srv6.MECHANISM,
				},
				{
					Cls:  cls.REMOTE,
					Type: vxlan.MECHANISM,
				},
				{
					Cls:  cls.REMOTE,
					Type: wireguard.MECHANISM,
				},
				{
					Cls:  cls.LOCAL,
					Type: kernel.MECHANISM,
				},
				{
					Cls:  cls.LOCAL,
					Type: memif.MECHANISM,
				},
			},
		}
	}
	samples := []struct {
		Name           string
		Request        *networkservice.NetworkServiceRequest
		Priorities     []string
		ExpectedResult []string
	}{
		{
			Name:           "Nil mechanisms",
			Request:        &networkservice.NetworkServiceRequest{},
			ExpectedResult: nil,
		},
		{
			Name:           "Empty mechanisms",
			Request:        &networkservice.NetworkServiceRequest{MechanismPreferences: []*networkservice.Mechanism{}},
			ExpectedResult: []string{},
		},
		{
			Name:           "No priority",
			Request:        request(),
			ExpectedResult: []string{srv6.MECHANISM, vxlan.MECHANISM, wireguard.MECHANISM, kernel.MECHANISM, memif.MECHANISM},
		},
		{
			Name:           "One priority",
			Request:        request(),
			Priorities:     []string{vxlan.MECHANISM},
			ExpectedResult: []string{vxlan.MECHANISM, srv6.MECHANISM, wireguard.MECHANISM, kernel.MECHANISM, memif.MECHANISM},
		},
		{
			Name:           "Multi priorities",
			Request:        request(),
			Priorities:     []string{kernel.MECHANISM, wireguard.MECHANISM, srv6.MECHANISM},
			ExpectedResult: []string{kernel.MECHANISM, wireguard.MECHANISM, srv6.MECHANISM, vxlan.MECHANISM, memif.MECHANISM},
		},
		{
			Name:           "Not supported mechanism in priority list",
			Request:        request(),
			Priorities:     []string{"NOT_SUPPORTED", vxlan.MECHANISM},
			ExpectedResult: []string{vxlan.MECHANISM, srv6.MECHANISM, wireguard.MECHANISM, kernel.MECHANISM, memif.MECHANISM},
		},
	}

	for _, s := range samples {
		sample := s
		t.Run(sample.Name, func(t *testing.T) {
			c := mechanismpriority.NewClient(sample.Priorities...)
			req := sample.Request
			_, err := c.Request(context.Background(), req)
			require.NoError(t, err)

			require.Equal(t, len(sample.ExpectedResult), len(req.GetMechanismPreferences()))
			for i := range req.GetMechanismPreferences() {
				require.Equal(t, sample.ExpectedResult[i], req.GetMechanismPreferences()[i].GetType())
			}
		})
	}
}
