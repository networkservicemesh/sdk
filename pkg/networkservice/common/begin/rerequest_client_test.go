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

package begin_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
)

func TestReRequestClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := chain.NewNetworkServiceClient(
		begin.NewClient(),
	)

	connOriginal, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, connOriginal)

	conn := connOriginal.Clone()
	conn.Context = &networkservice.ConnectionContext{
		IpContext: &networkservice.IPContext{
			SrcIpAddrs: []string{"10.0.0.1/32"},
		},
		EthernetContext: &networkservice.EthernetContext{
			SrcMac: "00:00:00:00:00:00",
		},
		DnsContext: &networkservice.DNSContext{
			Configs: []*networkservice.DNSConfig{
				{
					DnsServerIps: []string{"1.1.1.1"},
				},
			},
		},
		ExtraContext: map[string]string{"foo": "bar"},
	}
	conn.Mechanism = kernel.New("")
	conn.Labels = map[string]string{"foo": "bar"}

	connReturned, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: conn,
	})

	require.NoError(t, err)
	require.NotNil(t, connReturned)
	require.Equal(t, connOriginal.GetMechanism(), connReturned.GetMechanism())
	require.True(t, proto.Equal(conn.GetContext(), connReturned.GetContext()))
	require.Equal(t, connOriginal.GetLabels(), connReturned.GetLabels())
}
