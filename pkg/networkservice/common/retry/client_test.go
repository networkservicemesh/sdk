// Copyright (c) 2021 Cisco and/or its affiliates.
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

package retry_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/retry"
)

type remoteSideClient struct {
	delay            time.Duration
	failRequestCount int32
	failCloseCount   int32
}

func (c *remoteSideClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if atomic.AddInt32(&c.failRequestCount, -1) == -1 {
		return next.Client(ctx).Request(ctx, request, opts...)
	}

	clock.FromContext(ctx).Sleep(c.delay)
	return nil, errors.New("cannot connect")
}

func (c *remoteSideClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if atomic.AddInt32(&c.failCloseCount, -1) == -1 {
		return next.Client(ctx).Close(ctx, conn, opts...)
	}

	clock.FromContext(ctx).Sleep(c.delay)
	return nil, errors.New("cannot connect")
}

func Test_RetryClient_Request(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var counter = new(count.Client)

	var client = chain.NewNetworkServiceClient(
		chain.NewNetworkServiceClient(
			retry.NewClient(retry.WithInterval(time.Millisecond*10), retry.WithTryTimeout(time.Second/30)),
			counter,
			&remoteSideClient{
				delay:            time.Millisecond * 10,
				failRequestCount: 5,
			},
		),
	)

	var _, err = client.Request(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, 6, counter.Requests())
	require.Equal(t, 0, counter.Closes())
}

func Test_RetryClient_Request_ContextHasCorrectDeadline(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	clockMock.SetSpeed(0)

	ctx = clock.WithClock(ctx, clockMock)

	expectedDeadline := clockMock.Now().Add(time.Hour)

	var client = chain.NewNetworkServiceClient(
		retry.NewClient(retry.WithTryTimeout(time.Hour)),
		checkcontext.NewClient(t, func(t *testing.T, c context.Context) {
			v, ok := c.Deadline()
			require.True(t, ok)
			require.Equal(t, expectedDeadline, v)
		}),
	)

	var _, err = client.Request(ctx, nil)
	require.NoError(t, err)
}

func Test_RetryClient_Close_ContextHasCorrectDeadline(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	clockMock.SetSpeed(0)

	ctx = clock.WithClock(ctx, clockMock)

	expectedDeadline := clockMock.Now().Add(time.Hour)

	var client = chain.NewNetworkServiceClient(
		retry.NewClient(retry.WithTryTimeout(time.Hour)),
		checkcontext.NewClient(t, func(t *testing.T, c context.Context) {
			v, ok := c.Deadline()
			require.True(t, ok)
			require.Equal(t, expectedDeadline, v)
		}),
	)

	var _, err = client.Close(ctx, nil)
	require.NoError(t, err)
}

func Test_RetryClient_Close(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var counter = new(count.Client)

	var client = chain.NewNetworkServiceClient(
		retry.NewClient(
			retry.WithInterval(time.Millisecond*10),
			retry.WithTryTimeout(time.Second/30),
		),
		counter,
		&remoteSideClient{
			delay:          time.Millisecond * 10,
			failCloseCount: 5,
		},
	)

	var _, err = client.Close(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, 0, counter.Requests())
	require.Equal(t, 6, counter.Closes())
}
