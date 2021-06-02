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

package heal_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func TestHealClient_FindTest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nsmgrCtx, nsmgrCancel := context.WithCancel(ctx)
	defer nsmgrCancel()

	nodeConfig := []*sandbox.NodeConfig{{
		NsmgrCtx: nsmgrCtx,
	}}

	builder := sandbox.NewBuilder(t).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		SetContext(ctx).
		SetCustomConfig(nodeConfig)
	domain := builder.Build()

	// 1. Create NS, NSE find clients
	findCtx, findCancel := context.WithCancel(ctx)

	nsStream, err := domain.Nodes[0].NSRegistryClient.Find(findCtx, &registry.NetworkServiceQuery{
		NetworkService: new(registry.NetworkService),
		Watch:          true,
	})
	require.NoError(t, err)

	nseStream, err := domain.Nodes[0].EndpointRegistryClient.Find(findCtx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
		Watch:                  true,
	})
	require.NoError(t, err)

	// 2. Restart NSMgr
	nsmgrURL := grpcutils.URLToTarget(domain.Nodes[0].NSMgr.URL)

	nsmgrCancel()
	require.Eventually(t, grpcutils.CheckURLFree(nsmgrURL), time.Second, 100*time.Millisecond)

	mgr, resources := builder.NewNSMgr(ctx, domain.Nodes[0], nsmgrURL, domain.Registry.URL, sandbox.GenerateTestToken)
	domain.AddResources(resources)

	// 3. Register new NS, NSE
	_, err = mgr.NetworkServiceRegistryServer().Register(ctx, &registry.NetworkService{
		Name: "ns",
	})
	require.NoError(t, err)

	_, err = mgr.NetworkServiceEndpointRegistryServer().Register(ctx, &registry.NetworkServiceEndpoint{
		Name: "nse",
		Url:  "tcp://0.0.0.0",
	})
	require.NoError(t, err)

	// 4. Validate NS, NSE streams working
	ns, err := nsStream.Recv()
	require.NoError(t, err)
	require.Equal(t, "ns", ns.Name)

	nse, err := nseStream.Recv()
	require.NoError(t, err)
	require.Equal(t, "nse", nse.Name)

	// 5. Close NS, NSE streams
	findCancel()

	// 6. Validate NS, NSE streams closed
	timer := time.AfterFunc(100*time.Millisecond, t.FailNow)

	_, err = nsStream.Recv()
	require.Error(t, err)

	_, err = nseStream.Recv()
	require.Error(t, err)

	timer.Stop()
}
