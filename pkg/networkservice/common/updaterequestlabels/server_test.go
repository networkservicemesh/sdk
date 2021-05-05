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

package updaterequestlabels_test

import (
	"context"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updaterequestlabels"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
)

func TestUpdateLabels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	candidates := &discover.NetworkServiceCandidates{
		Endpoints: []*registry.NetworkServiceEndpoint{
			{Name: "nse-name1"},
			{Name: "nse-name2"},
		},
		DestLabels: []map[string]string{
			{"app": "firewall"},
			{"app": "vpn"},
		},
	}

	server := next.NewNetworkServiceServer(
		updaterequestlabels.NewServer(),
		checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
			require.Equal(t, candidates.DestLabels[0], request.Connection.Labels)
		}),
	)

	ctx = discover.WithCandidates(ctx, candidates)

	initialLabels := map[string]string{"not": "empty"}
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkServiceEndpointName: candidates.Endpoints[0].Name,
			Labels:                     initialLabels,
		},
	}
	conn, err := server.Request(ctx, request)
	require.NoError(t, err)
	require.Equal(t, initialLabels, conn.Labels)
}

func TestUpdateLabels_WithoutLabels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	candidates := &discover.NetworkServiceCandidates{
		Endpoints: []*registry.NetworkServiceEndpoint{
			{Name: "nse-name1"},
			{Name: "nse-name2"},
		},
	}

	initialLabels := map[string]string{"not": "empty"}

	server := next.NewNetworkServiceServer(
		updaterequestlabels.NewServer(),
		checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
			require.Equal(t, initialLabels, request.Connection.Labels)
		}),
	)

	ctx = discover.WithCandidates(ctx, candidates)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkServiceEndpointName: candidates.Endpoints[0].Name,
			Labels:                     initialLabels,
		},
	}
	conn, err := server.Request(ctx, request)
	require.NoError(t, err)
	require.Equal(t, initialLabels, conn.Labels)
}
