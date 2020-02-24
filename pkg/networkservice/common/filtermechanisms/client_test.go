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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/srv6"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/filtermechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type testNetworkServiceClient struct {
	t    *testing.T
	want []*networkservice.Mechanism
}

func (c *testNetworkServiceClient) Request(_ context.Context, in *networkservice.NetworkServiceRequest, _ ...grpc.CallOption) (*networkservice.Connection, error) {
	assert.Equal(c.t, c.want, in.GetMechanismPreferences())
	return in.GetConnection(), nil
}

func (c *testNetworkServiceClient) Close(context.Context, *networkservice.Connection, ...grpc.CallOption) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

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

var clientTestData = []struct {
	name string
	ctx  context.Context
	want []*networkservice.Mechanism
}{
	{
		"clientUrl has unix type",
		clienturl.WithClientURL(context.Background(), &url.URL{
			Scheme: "unix",
			Path:   "/var/run/nse-1.sock",
		}),
		[]*networkservice.Mechanism{
			{
				Cls:  cls.LOCAL,
				Type: memif.MECHANISM,
			},
			{
				Cls:  cls.LOCAL,
				Type: kernel.MECHANISM,
			},
		},
	},
	{
		"clientUrl has non-unix type",
		clienturl.WithClientURL(context.Background(), &url.URL{
			Scheme: "ipv4",
			Path:   "192.168.0.1",
		}),
		[]*networkservice.Mechanism{
			{
				Cls:  cls.REMOTE,
				Type: srv6.MECHANISM,
			},
			{
				Cls:  cls.REMOTE,
				Type: vxlan.MECHANISM,
			},
		},
	},
}

func Test_filterMechanismsClient_Request(t *testing.T) {
	for _, data := range clientTestData {
		test := data
		t.Run(test.name, func(t *testing.T) {
			testClientRequest(test.ctx, test.want, t)
		})
	}
}

func testClientRequest(ctx context.Context, want []*networkservice.Mechanism, t *testing.T) {
	client := next.NewNetworkServiceClient(filtermechanisms.NewClient(), &testNetworkServiceClient{t: t, want: want})
	_, _ = client.Request(ctx, request())
}
