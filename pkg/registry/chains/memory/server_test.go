// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
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

// Package memory_test define a tests for registry chain based on memory chain elements.
package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/api/pkg/api/registry"

	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"

	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func Test_RegistryMemory_ShouldSetDefaultPayload(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		Build()

	nsrc := registryclient.NewNetworkServiceRegistryClient(ctx,
		registryclient.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
		registryclient.WithClientURL(domain.Registry.URL))

	ns, err := nsrc.Register(ctx, &registry.NetworkService{
		Name: "ns-1",
	})
	require.NoError(t, err)

	nsrfc, err := nsrc.Find(ctx, &registry.NetworkServiceQuery{NetworkService: ns})
	require.NoError(t, err)

	nsResp, err := nsrfc.Recv()
	require.NoError(t, err)

	require.Equal(t, payload.IP, nsResp.GetNetworkService().GetPayload())

	require.NoError(t, ctx.Err())
}
