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

package selectservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/selectservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
)

const (
	svc1 = "service-1"
	svc2 = "service-2"
)

func TestCallNext(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var srvCounter, nextCounter atomic.Int32
	s := chain.NewNetworkServiceServer(
		selectservice.NewServer(map[string]networkservice.NetworkServiceServer{
			svc1: checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
				srvCounter.Inc()
			}),
			svc2: null.NewServer(),
		}),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			nextCounter.Inc()
		}),
	)

	conn, err := s.Request(ctx, &networkservice.NetworkServiceRequest{Connection: &networkservice.Connection{
		NetworkService: svc1,
	}})
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(1), srvCounter.Load())
	require.Equal(t, int32(1), nextCounter.Load())
}

func TestMissingService(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var nextCounter atomic.Int32
	s := chain.NewNetworkServiceServer(
		selectservice.NewServer(map[string]networkservice.NetworkServiceServer{
			svc2: null.NewServer(),
		}),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			nextCounter.Inc()
		}),
	)

	conn, err := s.Request(ctx, &networkservice.NetworkServiceRequest{Connection: &networkservice.Connection{
		NetworkService: svc1,
	}})
	require.Error(t, err)
	require.Nil(t, conn)
	require.Equal(t, int32(0), nextCounter.Load())
}
