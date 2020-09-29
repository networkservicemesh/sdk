// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package swap_test

import (
	"context"
	"net"
	"net/url"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/swap"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

func TestSwapIPServer_Request(t *testing.T) {
	const localIP = "127.0.0.1"
	const remoteIP = "172.16.1.1"
	const externalIP = "180.16.1.1"
	s := next.NewNetworkServiceServer(
		swap.NewServer(net.ParseIP(externalIP)),
		checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
			for _, m := range request.MechanismPreferences {
				require.Equal(t, m.GetParameters()[common.SrcIP], externalIP)
			}
			request.GetConnection().Mechanism = request.MechanismPreferences[0].Clone()
			request.GetConnection().Mechanism.GetParameters()[common.DstIP] = localIP
			require.False(t, interdomain.Is(request.GetConnection().NetworkServiceEndpointName))
			require.False(t, interdomain.Is(request.GetConnection().NetworkService))
		}))

	ctx := clienturlctx.WithClientURL(context.Background(), &url.URL{Scheme: "tcp", Host: remoteIP + ":5001"})
	response, err := s.Request(ctx, &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Cls: cls.REMOTE,
				Parameters: map[string]string{
					common.SrcIP: localIP,
				},
			},
		},
		Connection: &networkservice.Connection{
			NetworkService:             "my-ns1@remote_domain",
			NetworkServiceEndpointName: "my-nse1@remote_domain",
		},
	})
	require.NoError(t, err)
	require.Equal(t, response.Mechanism.Parameters[common.DstIP], remoteIP)
	require.True(t, interdomain.Is(response.NetworkServiceEndpointName))
	require.True(t, interdomain.Is(response.NetworkService))
}
