// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

package setregistrationtime_test

import (
	"context"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/setregistrationtime"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/metadata"
)

func testNSE() *registry.NetworkServiceEndpoint {
	return &registry.NetworkServiceEndpoint{
		Name: "nse",
		Url:  "tcp://0.0.0.0",
	}
}

func TestRegTimeServer_Register(t *testing.T) {
	s := next.NewNetworkServiceEndpointRegistryServer(
		metadata.NewNetworkServiceEndpointServer(),
		setregistrationtime.NewNetworkServiceEndpointRegistryServer(),
		memory.NewNetworkServiceEndpointRegistryServer(),
	)

	ctx := context.Background()
	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	// 1. Register
	reg, err := s.Register(ctx, testNSE())
	require.NoError(t, err)
	require.NotNil(t, reg.GetInitialRegistrationTime())
	require.True(t, clockMock.Now().Equal(reg.GetInitialRegistrationTime().AsTime().Local()))
	registeredNse := reg.Clone()

	// 2. Find
	nses := find(t, s, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.Len(t, nses, 1)
	require.True(t, proto.Equal(nses[0].GetInitialRegistrationTime(), registeredNse.GetInitialRegistrationTime()))

	// 3.1 Refresh
	reg, err = s.Register(ctx, reg.Clone())
	require.NoError(t, err)
	require.NotNil(t, reg.GetInitialRegistrationTime())
	require.True(t, proto.Equal(reg.GetInitialRegistrationTime(), registeredNse.GetInitialRegistrationTime()))

	// 3.2 Refresh with empty field
	regClone := reg.Clone()
	regClone.InitialRegistrationTime = nil
	clockMock.Add(time.Second)
	reg, err = s.Register(ctx, regClone)
	require.NoError(t, err)
	require.NotNil(t, reg.GetInitialRegistrationTime())
	require.True(t, proto.Equal(reg.GetInitialRegistrationTime(), registeredNse.GetInitialRegistrationTime()))

	// 4. Unregister
	_, err = s.Unregister(ctx, reg.Clone())
	require.NoError(t, err)

	// 5. Register again
	clockMock.Add(time.Second * 3)
	reg, err = s.Register(ctx, testNSE())
	require.NoError(t, err)
	require.NotNil(t, reg.GetInitialRegistrationTime())
	require.True(t, clockMock.Now().Equal(reg.GetInitialRegistrationTime().AsTime().Local()))
}

func find(t *testing.T, mem registry.NetworkServiceEndpointRegistryServer, query *registry.NetworkServiceEndpointQuery) (nses []*registry.NetworkServiceEndpoint) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch := make(chan *registry.NetworkServiceEndpointResponse)
	go func() {
		defer close(ch)
		require.NoError(t, mem.Find(query, streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch)))
	}()

	for nseResp := range ch {
		nses = append(nses, nseResp.GetNetworkServiceEndpoint())
	}

	return nses
}
