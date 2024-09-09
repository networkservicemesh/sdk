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

package replacelabels_test

import (
	"context"
	"testing"

	"go.uber.org/goleak"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/passthrough/replacelabels"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkclose"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func TestClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	chainLabels := map[string]string{"1": "A", "2": "B"}
	client := chain.NewNetworkServiceClient(
		metadata.NewClient(),
		replacelabels.NewClient(chainLabels),
		checkrequest.NewClient(t, func(t *testing.T, r *networkservice.NetworkServiceRequest) {
			require.Equal(t, chainLabels, r.GetConnection().GetLabels())
		}),
		checkclose.NewClient(t, func(t *testing.T, c *networkservice.Connection) {
			require.Equal(t, chainLabels, c.GetLabels())
		}),
	)

	// Create the request with any labels
	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{Id: "nsc-1", Labels: map[string]string{"3": "C"}},
	}
	conn, err := client.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"3": "C"}, conn.GetLabels())

	_, err = client.Close(context.Background(), conn)
	require.NoError(t, err)
}
