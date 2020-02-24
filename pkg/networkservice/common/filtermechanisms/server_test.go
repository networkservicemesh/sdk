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

package filtermechanisms_test

import (
	"context"
	"net"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/srv6"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/filtermechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type testNetworkServiceServer struct {
	t    *testing.T
	want []*networkservice.Mechanism
}

func (c *testNetworkServiceServer) Request(_ context.Context, in *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	assert.Equal(c.t, c.want, in.GetMechanismPreferences())
	return in.GetConnection(), nil
}

func (c *testNetworkServiceServer) Close(context.Context, *networkservice.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

var serverTestData = []struct {
	name string
	ctx  context.Context
	want []*networkservice.Mechanism
}{
	{
		"clientUrl has unix type",
		peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.UnixAddr{
				Name: "/var/run/nse-1.sock",
				Net:  "unix",
			},
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
		peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.IPAddr{
				IP: net.IP{192, 168, 0, 1},
			},
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

func Test_filterMechanismsServer_Request(t *testing.T) {
	for _, data := range serverTestData {
		test := data
		t.Run(test.name, func(t *testing.T) {
			testServerRequest(test.ctx, test.want, t)
		})
	}
}

func testServerRequest(ctx context.Context, want []*networkservice.Mechanism, t *testing.T) {
	client := next.NewNetworkServiceServer(filtermechanisms.NewServer(), &testNetworkServiceServer{t: t, want: want})
	_, _ = client.Request(ctx, request())
}
