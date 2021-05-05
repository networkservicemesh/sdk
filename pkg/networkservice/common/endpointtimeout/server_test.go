// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package endpointtimeout_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/endpointtimeout"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
)

const (
	testWait = 50 * time.Millisecond
	testTick = testWait / 100
)

func TestEndpointTimeout_NoRequests(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clockMock := clockmock.New()
	ctx = clock.WithClock(ctx, clockMock)

	timeout := time.Hour
	var flag atomic.Bool

	_ = endpointtimeout.NewServer(ctx, endpointtimeout.WithTimeout(timeout), endpointtimeout.WithNotify(func() {
		flag.Store(true)
	}))

	clockMock.Add(timeout - 1)
	require.Never(t, flag.Load, testWait, testTick)

	clockMock.Add(1)
	require.Eventually(t, flag.Load, testWait, testTick)
}

func TestEndpointTimeout_Refresh(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clockMock := clockmock.New()
	ctx = clock.WithClock(ctx, clockMock)

	timeout := time.Hour
	var flag atomic.Bool

	server := endpointtimeout.NewServer(ctx, endpointtimeout.WithTimeout(timeout), endpointtimeout.WithNotify(func() {
		flag.Store(true)
	}))

	clockMock.Add(timeout - 1)
	conn, err := server.Request(ctx, &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)
	clockMock.Add(1)
	require.Never(t, flag.Load, testWait, testTick)

	_, err = server.Close(ctx, conn) // Closes shouldn't affect timeout
	require.NoError(t, err)
	clockMock.Add(timeout - 1)
	require.Eventually(t, flag.Load, testWait, testTick)
}

func TestEndpointTimeout_ContextCancel(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clockMock := clockmock.New()
	ctx = clock.WithClock(ctx, clockMock)

	timeout := time.Hour
	var flag atomic.Bool

	_ = endpointtimeout.NewServer(ctx, endpointtimeout.WithTimeout(timeout), endpointtimeout.WithNotify(func() {
		flag.Store(true)
	}))

	cancel()
	runtime.Gosched()
	clockMock.Add(timeout)
	require.Never(t, flag.Load, testWait, testTick)
}
