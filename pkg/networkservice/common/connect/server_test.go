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

package connect

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/testnse"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
)

func TokenGenerator(peerAuthInfo credentials.AuthInfo) (token string, expireTime time.Time, err error) {
	return "TestToken", time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC), nil
}

type dummyConn struct {
}

func (d *dummyConn) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	return errors.Errorf("error is test")
}

func (d *dummyConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, errors.Errorf("error is test")
}

var dummyDialer = WithDialer(func(ctx context.Context, target string, opts ...grpc.DialOption) (conn grpc.ClientConnInterface, cancelFunc func(), err error) {
	return &dummyConn{}, func() {}, nil
})

func TestConnectServerShouldNotPanicOnRequest(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	listenOn := &url.URL{Scheme: "tcp", Path: "127.0.0.1:0"}

	nse, nseSrv, _ := testnse.NewNSE(ctx, listenOn, func(request *networkservice.NetworkServiceRequest) {
		request.Connection.Labels = map[string]string{"ok": "all is ok"}
	})
	defer nseSrv.Stop()

	require.NotPanics(t, func() {
		s := NewServer(func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
			return adapters.NewServerToClient(nse)
		}, dummyDialer)
		conn, err := s.Request(clienturl.WithClientURL(context.Background(), &url.URL{}), &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id: "1",
			},
		})
		require.Nil(t, err)
		require.NotNil(t, conn)
		require.Equal(t, "all is ok", conn.Labels["ok"])
	})
}

func TestDialFactoryRequest(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	listenOn := &url.URL{Scheme: "tcp", Path: "127.0.0.1:0"}

	_, nseSrv, _ := testnse.NewNSE(ctx, listenOn, func(request *networkservice.NetworkServiceRequest) {
		request.Connection.Labels = map[string]string{"ok": "all is ok"}
	})
	defer nseSrv.Stop()

	require.NotPanics(t, func() {
		s := NewServer(client.NewClientFactory("nsc", nil, TokenGenerator),
			WithDialOptionFactory(func(ctx context.Context, clientURL *url.URL) []grpc.DialOption {
				return []grpc.DialOption{grpc.WithInsecure()}
			}))
		ctx := clienturl.WithClientURL(context.Background(), &url.URL{Scheme: "tcp", Path: listenOn.Path})
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
