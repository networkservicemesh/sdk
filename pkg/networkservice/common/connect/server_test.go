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
	"fmt"
	"io/ioutil"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

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

type nseTest struct {
	ctx      context.Context
	cancel   context.CancelFunc
	listenOn *url.URL
	nse      networkservice.NetworkServiceServer
	nseSrv   *grpc.Server
}

func (nseT *nseTest) Stop() {
	nseT.cancel()
	nseT.nseSrv.Stop()
}

func (nseT *nseTest) Setup() {
	logrus.SetOutput(ioutil.Discard)
	nseT.ctx, nseT.cancel = context.WithTimeout(context.Background(), 5*time.Second)
	nseT.listenOn = &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}

	nseT.nse, nseT.nseSrv, _ = testnse.NewNSE(nseT.ctx, nseT.listenOn, func(request *networkservice.NetworkServiceRequest) {
		request.Connection.Context = &networkservice.ConnectionContext{
			ExtraContext: map[string]string{"ok": "all is ok"},
		}
	})
}

func (nseT *nseTest) newNSEContext(ctx context.Context) context.Context {
	return clienturl.WithClientURL(ctx, &url.URL{Scheme: "tcp", Host: nseT.listenOn.Host})
}

func TestConnectServerShouldNotPanicOnRequest(t *testing.T) {
	defer goleak.VerifyNone(t)

	nseT := &nseTest{}
	nseT.Setup()
	defer nseT.Stop()

	t.Run("Check Request", func(t *testing.T) {
		require.NotPanics(t, func() {
			serverCtx, serverCancel := context.WithCancel(context.Background())
			defer serverCancel()
			s := NewServer(serverCtx, func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
				return adapters.NewServerToClient(nseT.nse)
			}, grpc.WithInsecure())
			clientURLCtx := nseT.newNSEContext(context.Background())
			conn, err := s.Request(clientURLCtx, &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Id: "1",
				},
			})
			require.Nil(t, err)
			require.NotNil(t, conn)
			require.Equal(t, "all is ok", conn.GetContext().GetExtraContext()["ok"])
			_, err = s.Close(clientURLCtx, &networkservice.Connection{
				Id: "1",
			})
			require.Nil(t, err)
		})
	})
	t.Run("Close Id", func(t *testing.T) {
		require.NotPanics(t, func() {
			serverCtx, serverCancel := context.WithCancel(context.Background())
			defer serverCancel()
			s := NewServer(serverCtx, func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
				return adapters.NewServerToClient(nseT.nse)
			}, grpc.WithInsecure())
			clientURLCtx := nseT.newNSEContext(context.Background())
			conn, err := s.Request(clientURLCtx, &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Id: "1",
				},
			})
			require.Nil(t, err)
			require.NotNil(t, conn)
			require.Equal(t, "all is ok", conn.GetContext().GetExtraContext()["ok"])

			// Do not pass clientURL
			_, err = s.Close(context.Background(), &networkservice.Connection{
				Id: "1",
			})
			require.Nil(t, err)
		})
	})
	t.Run("Check no clientURL", func(t *testing.T) {
		require.NotPanics(t, func() {
			serverCtx, serverCancel := context.WithCancel(context.Background())
			defer serverCancel()
			s := NewServer(serverCtx, func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
				return adapters.NewServerToClient(nseT.nse)
			}, grpc.WithInsecure())

			conn, err := s.Request(context.Background(), &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Id: "1",
				},
			})
			require.NotNil(t, err)
			require.Nil(t, conn)
		})
	})
	t.Run("Request without client URL", func(t *testing.T) {
		require.NotPanics(t, func() {
			serverCtx, serverCancel := context.WithCancel(context.Background())
			defer serverCancel()
			s := NewServer(serverCtx, func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
				return adapters.NewServerToClient(nseT.nse)
			}, grpc.WithInsecure())
			clientURLCtx := nseT.newNSEContext(context.Background())
			conn, err := s.Request(clientURLCtx, &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Id: "1",
				},
			})
			require.Nil(t, err)
			require.NotNil(t, conn)
			require.Equal(t, "all is ok", conn.GetContext().GetExtraContext()["ok"])

			// Request again
			conn, err = s.Request(context.Background(), &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Id: "1",
				},
			})
			require.Nil(t, err)
			require.NotNil(t, conn)
			require.Equal(t, "all is ok", conn.GetContext().GetExtraContext()["ok"])

			// Do not pass clientURL
			_, err = s.Close(context.Background(), &networkservice.Connection{
				Id: "1",
			})
			require.Nil(t, err)
		})
	})
}

func TestParallelDial(t *testing.T) {
	defer goleak.VerifyNone(t)

	nseT := &nseTest{}
	nseT.Setup()
	defer nseT.Stop()

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	s := NewServer(serverCtx, client.NewClientFactory("nsc", nil, TokenGenerator), grpc.WithInsecure())
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		j := i
		ctx := nseT.newNSEContext(context.Background())
		go func() {
			defer wg.Done()
			for k := 0; k < 10; k++ {
				conn, err := s.Request(ctx, &networkservice.NetworkServiceRequest{
					Connection: &networkservice.Connection{
						Id: fmt.Sprintf("%d", j),
						Path: &networkservice.Path{
							PathSegments: []*networkservice.PathSegment{},
						},
					},
				})
				require.Nil(t, err)
				require.NotNil(t, conn)

				_, err = s.Close(ctx, conn)

				require.Nil(t, err)
			}
		}()
	}
	wg.Wait()
}
