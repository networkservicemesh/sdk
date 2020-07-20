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

// Package filtermechanisms_test provides a tests for package 'filtermechanisms'
package filtermechanisms_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/srv6"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/filtermechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
)

func request() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Cls:  cls.LOCAL,
				Type: memif.MECHANISM,
			},
			{
				Cls:  cls.LOCAL,
				Type: kernel.MECHANISM,
			},
			{
				Cls:  cls.REMOTE,
				Type: srv6.MECHANISM,
			},
			{
				Cls:  cls.REMOTE,
				Type: vxlan.MECHANISM,
			},
			{
				Cls:  "NOT_A_CLS",
				Type: "NOT_A_TYPE",
			},
		},
	}
}

func TestNewClient_FilterUnixType(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	ctx := clienturl.WithClientURL(context.Background(), &url.URL{
		Scheme: "unix",
		Path:   "/var/run/nse-1.sock",
	})
	client := next.NewNetworkServiceClient(
		filtermechanisms.NewClient(),
		checkrequest.NewClient(t, func(t *testing.T, serviceRequest *networkservice.NetworkServiceRequest) {
			expected := []*networkservice.Mechanism{
				{
					Cls:  cls.LOCAL,
					Type: memif.MECHANISM,
				},
				{
					Cls:  cls.LOCAL,
					Type: kernel.MECHANISM,
				},
			}
			assert.Equal(t, expected, serviceRequest.GetMechanismPreferences())
		}),
	)
	_, err := client.Request(ctx, request())
	assert.Nil(t, err)
}

func TestNewClient_FilterNonUnixType(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	ctx := clienturl.WithClientURL(context.Background(), &url.URL{
		Scheme: "ipv4",
		Path:   "192.168.0.1",
	})
	client := next.NewNetworkServiceClient(
		filtermechanisms.NewClient(),
		checkrequest.NewClient(t, func(t *testing.T, serviceRequest *networkservice.NetworkServiceRequest) {
			expected := []*networkservice.Mechanism{
				{
					Cls:  cls.REMOTE,
					Type: srv6.MECHANISM,
				},
				{
					Cls:  cls.REMOTE,
					Type: vxlan.MECHANISM,
				},
			}
			assert.Equal(t, expected, serviceRequest.GetMechanismPreferences())
		}),
	)
	_, err := client.Request(ctx, request())
	assert.Nil(t, err)
}
