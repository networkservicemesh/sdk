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

package timeout_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/timeout"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

const (
	tokenTimeout = 100 * time.Millisecond
	closeTimeout = 10 * tokenTimeout
)

func TestTimeoutServer_Request(t *testing.T) {
	connServer := &connectionsServer{
		connections: map[string]struct{}{},
	}

	client := chain.NewNetworkServiceClient(
		updatepath.NewClient("client"),
		updatetoken.NewClient(func(_ credentials.AuthInfo) (string, time.Time, error) {
			return "token", time.Now().Add(tokenTimeout), nil
		}),
		kernel.NewClient(),
		adapters.NewServerToClient(
			chain.NewNetworkServiceServer(
				updatepath.NewServer("server"),
				timeout.NewServer(nil),
				mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
					kernelmech.MECHANISM: connServer,
				}),
			),
		),
	)

	_, err := client.Request(context.TODO(), &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&connServer.size))

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&connServer.size) == 0
	}, closeTimeout, tokenTimeout, fmt.Sprintf("Not equal: \n"+
		"expected: %v\n"+
		"actual  : %v", 0, atomic.LoadInt32(&connServer.size)),
	)
}

type connectionsServer struct {
	connections map[string]struct{}
	size        int32
}

func (s *connectionsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	s.connections[request.GetConnection().GetId()] = struct{}{}
	atomic.StoreInt32(&s.size, int32(len(s.connections)))
	return next.Server(ctx).Request(ctx, request)
}

func (s *connectionsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	delete(s.connections, conn.GetId())
	atomic.StoreInt32(&s.size, int32(len(s.connections)))
	return next.Server(ctx).Close(ctx, conn)
}
