// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func TestHealClient_FindTest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nsmgrCtx, nsmgrCancel := context.WithCancel(ctx)
	defer nsmgrCancel()

	fwdName := sandbox.UniqueName("forwarder")
	domain := sandbox.NewBuilder(ctx, t).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, nodeNum int) {
			node.NewNSMgr(nsmgrCtx, sandbox.UniqueName("nsmgr"), nil, sandbox.GenerateTestToken, nsmgr.NewServer)
			node.NewForwarder(ctx, &registry.NetworkServiceEndpoint{
				Name:                fwdName,
				NetworkServiceNames: []string{"forwarder"},
				NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
					"forwarder": {
						Labels: map[string]string{
							"p2p": "true",
						},
					},
				},
			}, sandbox.GenerateTestToken)
		}).
		Build()

	// 1. Create NS, NSE find clients
	findCtx, findCancel := context.WithCancel(ctx)
	defer findCancel()

	nsRegistryClient := registryclient.NewNetworkServiceRegistryClient(ctx,
		registryclient.WithClientURL(sandbox.CloneURL(domain.Registry.URL)),
		registryclient.WithDialOptions(sandbox.DialOptions()...))

	_, err := nsRegistryClient.Find(findCtx, &registry.NetworkServiceQuery{
		NetworkService: new(registry.NetworkService),
		Watch:          true,
	})
	require.NoError(t, err)

}
