// Copyright (c) 2022 Cisco and/or its affiliates.
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

package cleanup_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/cleanup"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func TestCleanUp_CtxDone(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() { goleak.VerifyNone(t, goleak.IgnoreCurrent()) })
	chainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := new(count.Client)

	client := chain.NewNetworkServiceClient(
		begin.NewClient(),
		metadata.NewClient(),
		cleanup.NewClient(chainCtx),
		counter,
	)
	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{Id: "nsc-1"},
	}
	_, err := client.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, 1, counter.Requests())
	require.Equal(t, 0, counter.Closes())
	cancel()

	require.Eventually(t, func() bool {
		return counter.Closes() == 1
	}, time.Millisecond*100, time.Millisecond*10)
}

func TestCleanUp_Close(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() { goleak.VerifyNone(t, goleak.IgnoreCurrent()) })
	chainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := new(count.Client)

	client := chain.NewNetworkServiceClient(
		begin.NewClient(),
		metadata.NewClient(),
		cleanup.NewClient(chainCtx),
		counter,
	)
	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{Id: "nsc-1"},
	}
	conn, err := client.Request(context.Background(), req)
	require.NoError(t, err)

	_, _ = client.Close(context.Background(), conn)
	require.Equal(t, 1, counter.Closes())
	cancel()
	require.Never(t, func() bool {
		return counter.Closes() > 1
	}, time.Millisecond*100, time.Millisecond*10)
}

func TestCleanUp_Chan(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() { goleak.VerifyNone(t, goleak.IgnoreCurrent()) })
	chainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := new(count.Client)

	doneCh := make(chan struct{})
	client := chain.NewNetworkServiceClient(
		begin.NewClient(),
		metadata.NewClient(),
		cleanup.NewClient(chainCtx, cleanup.WithDoneChan(doneCh)),
		counter,
	)

	requestsNumber := 500
	for i := 0; i < requestsNumber; i++ {
		req := &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{Id: fmt.Sprintf("nsc-%v", i)},
		}
		_, err := client.Request(context.Background(), req)
		require.NoError(t, err)
	}

	cancel()
	<-doneCh

	require.Equal(t, counter.Closes(), requestsNumber)
}
