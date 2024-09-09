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

package roundrobin_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/roundrobin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/switchcase"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
)

const (
	nse1 = "nse-1"
	nse2 = "nse-2"
	ns   = "ns"
)

func TestSelectEndpointServer_CleanRequest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var hasBeenRequested bool
	s := next.NewNetworkServiceServer(
		roundrobin.NewServer(),
		switchcase.NewServer(
			&switchcase.ServerCase{
				Condition: func(_ context.Context, conn *networkservice.Connection) bool {
					return conn.GetNetworkServiceEndpointName() == nse1
				},
				Server: next.NewNetworkServiceServer(
					checkrequest.NewServer(t, func(_ *testing.T, request *networkservice.NetworkServiceRequest) {
						hasBeenRequested = true
						request.Connection.Labels[nse1] = nse1
					}),
					injecterror.NewServer(),
				),
			},
		),
	)

	ctx := discover.WithCandidates(context.Background(), []*registry.NetworkServiceEndpoint{
		{
			Name:                nse1,
			Url:                 "unix://" + nse1,
			NetworkServiceNames: []string{ns},
		},
		{
			Name:                nse2,
			Url:                 "unix://" + nse2,
			NetworkServiceNames: []string{ns},
		},
	}, &registry.NetworkService{Name: ns})

	conn, err := s.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Labels: map[string]string{nse2: nse2},
		},
	})
	require.NoError(t, err)

	require.True(t, hasBeenRequested)
	require.Equal(t, nse2, conn.GetNetworkServiceEndpointName())
	require.Equal(t, map[string]string{nse2: nse2}, conn.GetLabels())
}
