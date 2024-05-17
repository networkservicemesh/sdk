// Copyright (c) 2022-2024 Cisco and/or its affiliates.
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

package metrics_test

import (
	"context"
	"sync"

	"math/rand"
	"strconv"
	"testing"

	"go.uber.org/goleak"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/metrics"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

const (
	connectionCount = 1000
)

func TestMetrics_Concurrency(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var previousMetrics sync.Map
	server := chain.NewNetworkServiceServer(
		begin.NewServer(),
		metadata.NewServer(),
		updatepath.NewServer("testServer"),
		&metricsGeneratorServer{},
		metrics.NewServer(metrics.WithPreviousMetrics(&previousMetrics)),
	)

	wg := new(sync.WaitGroup)
	wg.Add(connectionCount)
	for i := 0; i < connectionCount; i++ {
		go func(i int) {
			defer wg.Done()
			req := &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{Id: "nsc-" + strconv.Itoa(i)},
			}
			conn, err := server.Request(context.Background(), req)
			require.NoError(t, err)
			defer func() {
				_, err = server.Close(context.Background(), conn)
				require.NoError(t, err)
			}()
		}(i)
	}
	wg.Wait()
	require.Equal(t, getMapSize(&previousMetrics), 0)
}

type metricsGeneratorServer struct{}

func (s *metricsGeneratorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	segment := request.GetConnection().GetPath().GetPathSegments()[0]
	if segment.Metrics == nil {
		segment.Metrics = make(map[string]string)
	}
	// Generate any random metric value
	// nolint:gosec
	segment.Metrics["testMetric"] = strconv.Itoa(rand.Intn(100))
	return next.Server(ctx).Request(ctx, request)
}

func (s *metricsGeneratorServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, connection)
}

func getMapSize(m *sync.Map) int {
	size := 0
	m.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}
