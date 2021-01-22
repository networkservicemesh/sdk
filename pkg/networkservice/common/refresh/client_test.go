// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package refresh_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

const (
	expireTimeout     = 100 * time.Millisecond
	eventuallyTimeout = expireTimeout
	tickTimeout       = 10 * time.Millisecond
	neverTimeout      = 5 * expireTimeout
	endpointName      = "endpoint-name"
)

func TestRefreshClient_StopRefreshAtClose(t *testing.T) {
	t.Skip("https://github.com/networkservicemesh/sdk/issues/237")
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cloneClient := &countClient{
		t: t,
	}
	client := chain.NewNetworkServiceClient(
		refresh.NewClient(ctx),
		cloneClient,
	)

	ctx = logger.WithLog(ctx)
	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
		},
	})
	require.NoError(t, err)
	require.Condition(t, cloneClient.validator(1))

	require.Eventually(t, cloneClient.validator(2), eventuallyTimeout, tickTimeout)

	_, err = client.Close(ctx, conn)
	require.NoError(t, err)

	require.Never(t, cloneClient.validator(3), neverTimeout, tickTimeout)
}

func TestRefreshClient_StopRefreshAtAnotherRequest(t *testing.T) {
	t.Skip("https://github.com/networkservicemesh/sdk/issues/260")
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	refreshClient := refresh.NewClient(ctx)
	cloneClient := &countClient{
		t: t,
	}
	client := chain.NewNetworkServiceClient(
		refreshClient,
		cloneClient,
	)

	ctx = logger.WithLog(ctx)
	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
		},
	})
	require.NoError(t, err)
	require.Condition(t, cloneClient.validator(1))

	require.Eventually(t, cloneClient.validator(2), eventuallyTimeout, tickTimeout)

	_, err = refreshClient.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: conn,
	})
	require.NoError(t, err)

	require.Never(t, cloneClient.validator(3), neverTimeout, tickTimeout)
}

type countClient struct {
	t     *testing.T
	count int32
}

func (c *countClient) validator(atLeast int32) func() bool {
	return func() bool {
		if count := atomic.LoadInt32(&c.count); count < atLeast {
			return false
		}
		return true
	}
}

func (c *countClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	request = request.Clone()
	conn := request.GetConnection()

	if atomic.AddInt32(&c.count, 1) == 1 {
		conn.NetworkServiceEndpointName = endpointName
	} else {
		require.Equal(c.t, endpointName, conn.NetworkServiceEndpointName)
	}

	setExpires(conn, expireTimeout)

	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *countClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func setExpires(conn *networkservice.Connection, expireTimeout time.Duration) {
	expireTime := time.Now().Add(expireTimeout)
	expires := &timestamp.Timestamp{
		Seconds: expireTime.Unix(),
		Nanos:   int32(expireTime.Nanosecond()),
	}
	conn.Path = &networkservice.Path{
		Index: 0,
		PathSegments: []*networkservice.PathSegment{
			{
				Expires: expires,
			},
		},
	}
}
