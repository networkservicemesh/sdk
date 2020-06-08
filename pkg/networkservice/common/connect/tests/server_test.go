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

package tests

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func TokenGenerator(peerAuthInfo credentials.AuthInfo) (token string, expireTime time.Time, err error) {
	return "TestToken", time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC), nil
}

func TestConnectServerShouldNotPanicOnRequest(t *testing.T) {
	defer goleak.VerifyNone(t)
	require.NotPanics(t, func() {
		s := connect.NewServer(client.NewClientFactory(t.Name(), nil, TokenGenerator, next.NewNetworkServiceClient()))
		_, _ = s.Request(clienturl.WithClientURL(context.Background(), &url.URL{}), nil)
	})
}

type testCon struct {
}

func (t *testCon) Read(b []byte) (n int, err error) {
	return len(b), nil
}

func (t *testCon) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (t *testCon) Close() error {
	return nil
}

func (t *testCon) LocalAddr() net.Addr {
	return &net.IPAddr{
		IP: net.IPv4(0, 0, 0, 1),
	}
}

func (t *testCon) RemoteAddr() net.Addr {
	return &net.IPAddr{
		IP: net.IPv4(0, 0, 0, 1),
	}
}

func (t *testCon) SetDeadline(_ time.Time) error {
	return nil
}

func (t *testCon) SetReadDeadline(_ time.Time) error {
	return nil
}

func (t *testCon) SetWriteDeadline(_ time.Time) error {
	return nil
}

type dummyNse struct {
	networkservice.NetworkServiceServer
	networkservice.MonitorConnectionServer
}

func (d *dummyNse) MonitorConnections(selector *networkservice.MonitorScopeSelector, server networkservice.MonitorConnection_MonitorConnectionsServer) error {
	// Just wait to end
	<-server.Context().Done()
	return nil
}

func (d *dummyNse) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	request.Connection.Labels = map[string]string{}
	return request.GetConnection(), nil
}

func (d *dummyNse) Close(ctx context.Context, connection *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func TestDialFactoryRequest(t *testing.T) {
	defer goleak.VerifyNone(t)

	dnse := &dummyNse{}

	require.NotPanics(t, func() {
		s := connect.NewServer(func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
			_ = cc.(*grpc.ClientConn).Close()
			return dnse
		},
			connect.WithDialOptionFactory(func(ctx context.Context, request *networkservice.NetworkServiceRequest, clientURL *url.URL) []grpc.DialOption {
				return []grpc.DialOption{grpc.WithInsecure(), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
					return &testCon{}, nil
				})}
			}))
		ctx := clienturl.WithClientURL(context.Background(), &url.URL{Scheme: "tcp", Path: "dummy"})
		conn, err := s.Request(ctx, &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id: "1",
				Path: &networkservice.Path{
					PathSegments: []*networkservice.PathSegment{},
				},
			},
		})
		require.Nil(t, err)
		require.NotNil(t, conn)

		_, err = s.Close(ctx, conn)

		require.Nil(t, err)
	})
}
