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

package swapip_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/swapip"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkresponse"
	"github.com/networkservicemesh/sdk/pkg/tools/fs"
)

//nolint:goconst
func TestSwapIPClient_Request(t *testing.T) {
	defer goleak.VerifyNone(t)

	p1 := filepath.Join(t.TempDir(), "map-ip-1.yaml")
	p2 := filepath.Join(t.TempDir(), "map-ip-2.yaml")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10000)
	defer cancel()

	err := os.WriteFile(p1, []byte(`172.16.2.10: 172.16.1.10`), os.ModePerm)
	require.NoError(t, err)

	err = os.WriteFile(p2, []byte(`172.16.2.100: 172.16.1.100`), os.ModePerm)
	require.NoError(t, err)

	ch1 := convertBytesChToMapCh(fs.WatchFile(ctx, p1))
	ch2 := convertBytesChToMapCh(fs.WatchFile(ctx, p2))

	testChain := next.NewNetworkServiceClient(
		/* Source side */
		checkresponse.NewClient(t, func(t *testing.T, c *networkservice.Connection) {
			require.Equal(t, "172.16.2.10", c.GetMechanism().GetParameters()[common.SrcIP])
			require.Equal(t, "", c.GetMechanism().GetParameters()[common.SrcOriginalIP])
			require.Equal(t, "172.16.2.100", c.GetMechanism().GetParameters()[common.DstIP])
			require.Equal(t, "172.16.2.100", c.GetMechanism().GetParameters()[common.DstOriginalIP])
		}),
		swapip.NewClient(ch1),
		checkrequest.NewClient(t, func(t *testing.T, r *networkservice.NetworkServiceRequest) {
			require.Equal(t, "172.16.1.10", r.GetConnection().GetMechanism().GetParameters()[common.SrcIP])
			require.Equal(t, "172.16.2.10", r.GetConnection().GetMechanism().GetParameters()[common.SrcOriginalIP])
		}),
		/* Destination side */
		checkresponse.NewClient(t, func(t *testing.T, c *networkservice.Connection) {
			require.Equal(t, "172.16.1.10", c.GetMechanism().GetParameters()[common.SrcIP])
			require.Equal(t, "172.16.2.10", c.GetMechanism().GetParameters()[common.SrcOriginalIP])
		}),
		swapip.NewClient(ch2),
		checkrequest.NewClient(t, func(t *testing.T, r *networkservice.NetworkServiceRequest) {
			r.Connection.Mechanism.Parameters[common.DstIP] = "172.16.2.100"
			r.Connection.Mechanism.Parameters[common.DstOriginalIP] = "172.16.2.100"
		}),
	)

	r := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{
					common.SrcIP: "172.16.2.10",
				},
			},
		},
	}

	time.Sleep(time.Second / 4)

	resp, err := testChain.Request(context.Background(), r)
	require.NoError(t, err)

	// refresh
	_, err = testChain.Request(ctx, &networkservice.NetworkServiceRequest{Connection: resp})
	require.NoError(t, err)
}
