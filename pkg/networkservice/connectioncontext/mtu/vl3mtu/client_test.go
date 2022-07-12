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

package vl3mtu_test

import (
	"context"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/mtu/vl3mtu"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func Test_vl3MtuClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := next.NewNetworkServiceClient(
		vl3mtu.NewClient(),
	)

	// Send the first Request
	resp1, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:      "1",
			Context: &networkservice.ConnectionContext{MTU: 2000},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint32(2000), resp1.GetContext().GetMTU())

	// Send request without MTU
	resp2, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "2",
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint32(2000), resp2.GetContext().GetMTU())

	// Send request with lower MTU
	resp3, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:      "3",
			Context: &networkservice.ConnectionContext{MTU: 1500},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint32(1500), resp3.GetContext().GetMTU())

	// Refresh the first connection
	resp1, err = client.Request(ctx, &networkservice.NetworkServiceRequest{Connection: resp1})

	require.NoError(t, err)
	require.Equal(t, uint32(1500), resp1.GetContext().GetMTU())
}
