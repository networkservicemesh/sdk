// Copyright (c) 2024 Cisco and/or its affiliates.
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

package begin_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

const (
	waitTime = time.Second
)

type waitServer struct {
	requestDone atomic.Int32
	closeDone   atomic.Int32
}

func (s *waitServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	time.Sleep(waitTime)
	s.requestDone.Add(1)
	return next.Server(ctx).Request(ctx, request)
}

func (s *waitServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	afterCh := time.After(time.Second)
	select {
	case <-ctx.Done():
		return &emptypb.Empty{}, nil
	case <-afterCh:
		s.closeDone.Add(1)
	}
	return next.Server(ctx).Close(ctx, connection)
}

func TestBeginWorksWithSmallTimeout(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})
	requestCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
	defer cancel()

	waitSrv := &waitServer{}
	server := next.NewNetworkServiceServer(
		begin.NewServer(),
		waitSrv,
	)

	request := testRequest("id")
	_, err := server.Request(requestCtx, request)
	require.EqualError(t, err, context.DeadlineExceeded.Error())
	require.Equal(t, int32(0), waitSrv.requestDone.Load())
	require.Eventually(t, func() bool {
		return waitSrv.requestDone.Load() == 1
	}, waitTime*2, time.Millisecond*500)

	closeCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
	defer cancel()
	_, err = server.Close(closeCtx, request.Connection)
	require.EqualError(t, err, context.DeadlineExceeded.Error())
	require.Equal(t, int32(0), waitSrv.closeDone.Load())
	require.Eventually(t, func() bool {
		return waitSrv.closeDone.Load() == 1
	}, waitTime*2, time.Millisecond*500)
}

func TestBeginHasExtendedTimeoutOnReselect(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})
	requestCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
	defer cancel()

	waitSrv := &waitServer{}
	server := next.NewNetworkServiceServer(
		begin.NewServer(),
		waitSrv,
	)

	request := testRequest("id")
	request.Connection.State = networkservice.State_RESELECT_REQUESTED

	_, err := server.Request(requestCtx, request)
	require.EqualError(t, err, context.DeadlineExceeded.Error())
	require.Equal(t, int32(0), waitSrv.requestDone.Load())
	require.Eventually(t, func() bool {
		return waitSrv.requestDone.Load() == 1
	}, waitTime*2, time.Millisecond*500)

	requestCtx, cancel = context.WithTimeout(context.Background(), time.Millisecond*200)
	defer cancel()
	request.Connection.State = networkservice.State_RESELECT_REQUESTED

	_, err = server.Request(requestCtx, request)
	require.EqualError(t, err, context.DeadlineExceeded.Error())
	require.Equal(t, int32(0), waitSrv.closeDone.Load())
	require.Eventually(t, func() bool {
		return waitSrv.closeDone.Load() == 1 && waitSrv.requestDone.Load() == 2
	}, waitTime*4, time.Millisecond*500)
}
