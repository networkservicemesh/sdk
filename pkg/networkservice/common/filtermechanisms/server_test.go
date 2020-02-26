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

type testNetworkServiceServer struct{}

func (s testNetworkServiceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	testData(ctx).mechanisms = request.GetMechanismPreferences()
	return next.Server(ctx).Request(ctx, request)
}

func (s testNetworkServiceServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

func TestNewServer_FilterUnixType(t *testing.T) {
	server := next.NewNetworkServiceServer(filtermechanisms.NewServer(), &testNetworkServiceServer{})
	ctx := withTestData(context.Background(), &TestData{})
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: &net.UnixAddr{
			Name: "/var/run/nse-1.sock",
			Net:  "unix",
		},
	})
	_, err := server.Request(ctx, request())
	assert.Nil(t, err)
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
	assert.Equal(t, expected, testData(ctx).mechanisms)
}

func TestNewServer_FilterNonUnixType(t *testing.T) {
	server := next.NewNetworkServiceServer(filtermechanisms.NewServer(), &testNetworkServiceServer{})
	ctx := withTestData(context.Background(), &TestData{})
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: &net.IPAddr{
			IP: net.IP{192, 168, 0, 1},
		},
	})
	_, err := server.Request(ctx, request())
	assert.Nil(t, err)
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
	assert.Equal(t, expected, testData(ctx).mechanisms)
}
