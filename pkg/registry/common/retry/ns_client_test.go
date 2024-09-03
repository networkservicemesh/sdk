// Copyright (c) 2021-2022 Cisco and/or its affiliates.
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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/registry/common/retry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/count"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
)

func TestNSRetryClient_Register(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	callCounter := &count.CallCounter{}
	counter := count.NewNetworkServiceRegistryClient(callCounter)

	client := chain.NewNetworkServiceRegistryClient(
		retry.NewNetworkServiceRegistryClient(
			context.Background(),
			retry.WithInterval(time.Millisecond*10),
			retry.WithTryTimeout(time.Second/30)),
		counter,
		injecterror.NewNetworkServiceRegistryClient(injecterror.WithRegisterErrorTimes(0, 1, 2, 3, 4)),
	)

	_, err := client.Register(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, 6, callCounter.Registers())
}

func TestNSRetryClient_Register_ContextHasCorrectDeadline(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	clockMock.SetSpeed(0)

	ctx = clock.WithClock(ctx, clockMock)

	expectedDeadline := clockMock.Now().Add(time.Hour)

	client := chain.NewNetworkServiceRegistryClient(
		retry.NewNetworkServiceRegistryClient(context.Background(), retry.WithTryTimeout(time.Hour)),
		checkcontext.NewNSClient(t, func(t *testing.T, c context.Context) {
			v, ok := c.Deadline()
			require.True(t, ok)
			require.Equal(t, expectedDeadline, v)
		}))

	_, err := client.Register(ctx, nil)
	require.NoError(t, err)
}

func TestNSRetryClient_Unregister_ContextHasCorrectDeadline(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	clockMock.SetSpeed(0)

	ctx = clock.WithClock(ctx, clockMock)

	expectedDeadline := clockMock.Now().Add(time.Hour)

	client := chain.NewNetworkServiceRegistryClient(
		retry.NewNetworkServiceRegistryClient(context.Background(), retry.WithTryTimeout(time.Hour)),
		checkcontext.NewNSClient(t, func(t *testing.T, c context.Context) {
			v, ok := c.Deadline()
			require.True(t, ok)
			require.Equal(t, expectedDeadline, v)
		}))

	_, err := client.Unregister(ctx, nil)
	require.NoError(t, err)
}

func TestNSRetryClient_Unregister(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	callCounter := &count.CallCounter{}
	counter := count.NewNetworkServiceRegistryClient(callCounter)

	client := chain.NewNetworkServiceRegistryClient(
		retry.NewNetworkServiceRegistryClient(
			context.Background(),
			retry.WithInterval(time.Millisecond*10),
			retry.WithTryTimeout(time.Second/30)),
		counter,
		injecterror.NewNetworkServiceRegistryClient(injecterror.WithUnregisterErrorTimes(0, 1, 2, 3, 4)),
	)

	_, err := client.Unregister(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, 6, callCounter.Unregisters())
}

func TestNSRetryClient_Find(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	callCounter := &count.CallCounter{}
	counter := count.NewNetworkServiceRegistryClient(callCounter)

	client := chain.NewNetworkServiceRegistryClient(
		retry.NewNetworkServiceRegistryClient(
			context.Background(),
			retry.WithInterval(time.Millisecond*10),
			retry.WithTryTimeout(time.Second/30)),
		counter,
		injecterror.NewNetworkServiceRegistryClient(injecterror.WithFindErrorTimes(0, 1, 2, 3, 4)),
	)

	_, err := client.Find(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, 6, callCounter.Finds())
}

func TestNSRetryClient_RegisterCompletesOnParentContextTimeout(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	callCounter := &count.CallCounter{}
	counter := count.NewNetworkServiceRegistryClient(callCounter)

	client := chain.NewNetworkServiceRegistryClient(
		retry.NewNetworkServiceRegistryClient(
			context.Background(),
			retry.WithInterval(time.Millisecond*10),
			retry.WithTryTimeout(time.Second/30)),
		counter,
		injecterror.NewNetworkServiceRegistryClient(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*55)
	defer cancel()

	_, err := client.Register(ctx, nil)
	require.Error(t, err)
	require.Greater(t, callCounter.Registers(), 0)
}

func TestNSRetryClient_UnregisterCompletesOnParentContextTimeout(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	callCounter := &count.CallCounter{}
	counter := count.NewNetworkServiceRegistryClient(callCounter)

	client := chain.NewNetworkServiceRegistryClient(
		retry.NewNetworkServiceRegistryClient(
			context.Background(),
			retry.WithInterval(time.Millisecond*10),
			retry.WithTryTimeout(time.Second/30)),
		counter,
		injecterror.NewNetworkServiceRegistryClient(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*55)
	defer cancel()

	_, err := client.Unregister(ctx, nil)
	require.Error(t, err)
	require.Greater(t, callCounter.Unregisters(), 0)
}

func TestNSRetryClient_FindCompletesOnParentContextTimeout(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	callCounter := &count.CallCounter{}
	counter := count.NewNetworkServiceRegistryClient(callCounter)

	client := chain.NewNetworkServiceRegistryClient(
		retry.NewNetworkServiceRegistryClient(
			context.Background(),
			retry.WithInterval(time.Millisecond*10),
			retry.WithTryTimeout(time.Second/30)),
		counter,
		injecterror.NewNetworkServiceRegistryClient(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*55)
	defer cancel()

	_, err := client.Find(ctx, nil)
	require.Error(t, err)
	require.Greater(t, callCounter.Finds(), 0)
}
