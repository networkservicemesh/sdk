// Copyright (c) 2023-2024 Cisco Systems, Inc.
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

package netsvcmonitor_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/netsvcmonitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
)

func Test_Netsvcmonitor_And_GroupOfSimilarNetworkServices(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})

	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	nsServer := memory.NewNetworkServiceRegistryServer()
	nseServer := memory.NewNetworkServiceEndpointRegistryServer()
	var counter count.Server

	_, _ = nsServer.Register(context.Background(), &registry.NetworkService{
		Name: "service-1",
	})

	_, _ = nseServer.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name:                "endpoint-1",
		NetworkServiceNames: []string{"service-1"},
	})

	server := chain.NewNetworkServiceServer(
		metadata.NewServer(),
		begin.NewServer(),
		netsvcmonitor.NewServer(
			testCtx,
			adapters.NetworkServiceServerToClient(nsServer),
			adapters.NetworkServiceEndpointServerToClient(nseServer),
		),
		&counter,
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:                         "1",
			NetworkService:             "service-1",
			NetworkServiceEndpointName: "endpoint-1",
		},
	}
	_, err := server.Request(testCtx, request)
	require.NoError(t, err)
	require.Equal(t, 0, counter.Closes())
	for i := 0; i < 10; i++ {
		_, err = nsServer.Register(context.Background(), &registry.NetworkService{
			Name: fmt.Sprintf("service-1%v", i),
			Matches: []*registry.Match{
				{
					SourceSelector: map[string]string{
						"color": "red",
					},
				},
			},
		})
		require.NoError(t, err)
	}

	require.Never(t, func() bool {
		return counter.Closes() > 0
	}, time.Millisecond*300, time.Millisecond*50)
}

func Test_NetsvcMonitor_ShouldNotLeakWithoutClose(t *testing.T) {
	testCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	t.Cleanup(func() {
		require.Eventually(t, func() bool {
			return goleak.Find(goleak.IgnoreAnyFunction("github.com/stretchr/testify/assert.Eventually")) == nil
		}, time.Second*2, time.Second/10)
		cancel()
	})
	nsServer := memory.NewNetworkServiceRegistryServer()
	nseServer := memory.NewNetworkServiceEndpointRegistryServer()
	var counter count.Server

	_, _ = nsServer.Register(context.Background(), &registry.NetworkService{
		Name: "service-1",
	})

	_, _ = nseServer.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name:                "endpoint-1",
		NetworkServiceNames: []string{"service-1"},
	})

	server := chain.NewNetworkServiceServer(
		metadata.NewServer(),
		begin.NewServer(),
		netsvcmonitor.NewServer(
			testCtx,
			adapters.NetworkServiceServerToClient(nsServer),
			adapters.NetworkServiceEndpointServerToClient(nseServer),
		),
		&counter,
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:                         "1",
			NetworkService:             "service-1",
			NetworkServiceEndpointName: "endpoint-1",
			Path: &networkservice.Path{
				PathSegments: []*networkservice.PathSegment{
					{
						Expires: timestamppb.New(time.Now().Add(time.Second)),
					},
				},
			},
		},
	}

	_, err := server.Request(testCtx, request)
	require.NoError(t, err)
}
