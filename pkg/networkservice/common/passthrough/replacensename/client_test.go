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

package replacensename_test

import (
	"context"
	"testing"

	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/passthrough/replacensename"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkclose"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func TestClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	client := chain.NewNetworkServiceClient(
		metadata.NewClient(),
		replacensename.NewClient(),
		&setNSENameClient{name: "nse-name1"},
		checkrequest.NewClient(t, func(t *testing.T, r *networkservice.NetworkServiceRequest) {
			require.Equal(t, "nse-name1", r.GetConnection().GetNetworkServiceEndpointName())
		}),
		checkclose.NewClient(t, func(t *testing.T, c *networkservice.Connection) {
			require.Equal(t, "nse-name1", c.GetNetworkServiceEndpointName())
		}),
	)
	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{Id: "nsc-1"},
	}
	conn, err := client.Request(context.Background(), req)
	require.NoError(t, err)

	// Change NetworkServiceEndpointName to another name
	conn.NetworkServiceEndpointName = "nse-name2"
	conn, err = client.Request(context.Background(), req)
	require.Equal(t, "nse-name2", conn.GetNetworkServiceEndpointName())
	require.NoError(t, err)

	_, err = client.Close(context.Background(), conn)
	require.NoError(t, err)
}

// setNSENameClient sets NetworkServiceEndpointName only if it is empty.
type setNSENameClient struct {
	name string
}

func (s *setNSENameClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if request.GetConnection().GetNetworkServiceEndpointName() == "" {
		request.GetConnection().NetworkServiceEndpointName = s.name
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (s *setNSENameClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
