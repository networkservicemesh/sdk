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

package defaultexpire_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/defaultexpire"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
)

func TestDefaultExpireNSEServer(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ctx = clock.WithClock(ctx, clockmock.New(ctx))

	var samples = []struct {
		name string
		nse  *registry.NetworkServiceEndpoint
	}{
		{
			name: "With NSE expiration",
			nse: &registry.NetworkServiceEndpoint{
				Name:                "nse",
				NetworkServiceNames: []string{"ns"},
				ExpirationTime:      timestamppb.New(clock.FromContext(ctx).Now().Add(time.Minute).Local()),
			},
		},
		{
			name: "Without NSE expiration",
			nse: &registry.NetworkServiceEndpoint{
				Name:                "nse",
				NetworkServiceNames: []string{"ns"},
			},
		},
	}

	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			// nolint:scopelint
			testDefaultExpireNSEServer(ctx, t, sample.nse)
		})
	}
}

func testDefaultExpireNSEServer(ctx context.Context, t *testing.T, nse *registry.NetworkServiceEndpoint) {
	s := next.NewNetworkServiceEndpointRegistryServer(
		defaultexpire.NewNetworkServiceEndpointRegistryServer(ctx, time.Hour),
	)

	registeredNSE, err := s.Register(ctx, nse.Clone())
	require.NoError(t, err)

	if nse.GetExpirationTime() != nil {
		require.Equal(t, nse.GetExpirationTime(), registeredNSE.GetExpirationTime())
	} else {
		require.Equal(t, clock.FromContext(ctx).Now().Local().Add(time.Hour), registeredNSE.GetExpirationTime().AsTime().Local())
	}
}
