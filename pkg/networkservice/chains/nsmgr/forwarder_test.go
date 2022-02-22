// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

package nsmgr_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func Test_RemoteForwarderShouldBeSelectedCorrectlyOnNSMgrRestart(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		Build()

	var expectedForwarderName string

	require.Len(t, domain.Nodes[0].Forwarders, 1)
	for k := range domain.Nodes[0].Forwarders {
		expectedForwarderName = k
	}

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	_, err := nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-ns",
	})
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "my-nse-1",
		NetworkServiceNames: []string{"my-ns"},
	}

	domain.Nodes[1].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest("my-ns")

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 6, len(conn.Path.PathSegments))
	require.Equal(t, expectedForwarderName, conn.GetPath().GetPathSegments()[2].Name)

	for i := 0; i < 10; i++ {
		request.Connection = conn.Clone()
		conn, err = nsc.Request(ctx, request.Clone())
		require.NoError(t, err)
		require.Equal(t, expectedForwarderName, conn.GetPath().GetPathSegments()[2].Name)

		domain.Nodes[0].NewForwarder(ctx, &registryapi.NetworkServiceEndpoint{
			Name:                sandbox.UniqueName(fmt.Sprintf("%v-forwarder", i)),
			NetworkServiceNames: []string{"forwarder"},
			NetworkServiceLabels: map[string]*registryapi.NetworkServiceLabels{
				"forwarder": {
					Labels: map[string]string{
						"p2p": "true",
					},
				},
			},
		}, sandbox.GenerateTestToken)

		domain.Nodes[0].NSMgr.Restart()
	}
}

func Test_LocalForwarderShouldBeSelectedCorrectlyOnNSMgrRestart(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		Build()

	var expectedForwarderName string

	require.Len(t, domain.Nodes[0].Forwarders, 1)
	for k := range domain.Nodes[0].Forwarders {
		expectedForwarderName = k
	}

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	_, err := nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-ns",
	})
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "my-nse-1",
		NetworkServiceNames: []string{"my-ns"},
	}

	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest("my-ns")

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 4, len(conn.Path.PathSegments))
	require.Equal(t, expectedForwarderName, conn.GetPath().GetPathSegments()[2].Name)

	for i := 0; i < 10; i++ {
		request.Connection = conn.Clone()
		conn, err = nsc.Request(ctx, request.Clone())
		require.NoError(t, err)
		require.Equal(t, expectedForwarderName, conn.GetPath().GetPathSegments()[2].Name)

		domain.Nodes[0].NewForwarder(ctx, &registryapi.NetworkServiceEndpoint{
			Name:                sandbox.UniqueName(fmt.Sprintf("%v-forwarder", i)),
			NetworkServiceNames: []string{"forwarder"},
			NetworkServiceLabels: map[string]*registryapi.NetworkServiceLabels{
				"forwarder": {
					Labels: map[string]string{
						"p2p": "true",
					},
				},
			},
		}, sandbox.GenerateTestToken)

		domain.Nodes[0].NSMgr.Restart()
	}
}
