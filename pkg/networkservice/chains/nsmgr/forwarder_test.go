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

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	registryadapter "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func Test_ForwarderShouldBeSelectedCorrectlyOnNSMgrRestart(t *testing.T) {
	var samples = []struct {
		name             string
		nodeNum          int
		pathSegmentCount int
	}{
		{
			name:             "Local",
			nodeNum:          0,
			pathSegmentCount: 4,
		},
		{
			name:             "Remote",
			nodeNum:          1,
			pathSegmentCount: 6,
		},
	}

	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			// nolint:scopelint
			testForwarderShouldBeSelectedCorrectlyOnNSMgrRestart(t, sample.nodeNum, sample.pathSegmentCount)
		})
	}
}

func testForwarderShouldBeSelectedCorrectlyOnNSMgrRestart(t *testing.T, nodeNum, pathSegmentCount int) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*40)
	logrus.SetLevel(logrus.TraceLevel)
	log.EnableTracing(true)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(nodeNum + 1).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		Build()

	var expectedForwarderName string

	require.Len(t, domain.Nodes[0].Forwarders, 1)
	for k := range domain.Nodes[0].Forwarders {
		expectedForwarderName = k
	}

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	_, err := nsRegistryClient.Register(ctx, &registryapi.NetworkService{
		Name: "my-ns",
	})
	require.NoError(t, err)

	nseReg := &registryapi.NetworkServiceEndpoint{
		Name:                "my-nse-1",
		NetworkServiceNames: []string{"my-ns"},
	}

	domain.Nodes[nodeNum].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest("my-ns")

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, pathSegmentCount, len(conn.Path.PathSegments))
	require.Equal(t, expectedForwarderName, conn.GetPath().GetPathSegments()[2].Name)

	for i := 0; i < 3; i++ {
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

		nseClient := registryadapter.NetworkServiceEndpointServerToClient(domain.Nodes[0].NSMgr.Nsmgr.NetworkServiceEndpointRegistryServer())
		request := &registryapi.NetworkServiceEndpointQuery{
			NetworkServiceEndpoint: &registryapi.NetworkServiceEndpoint{
				Name: expectedForwarderName,
				Url:  domain.Nodes[0].NSMgr.URL.String(),
			},
		}

		stream, _ := nseClient.Find(ctx, request)
		msg, _ := stream.Recv()

		for msg == nil {
			stream, _ = nseClient.Find(ctx, request)
			msg, _ = stream.Recv()
		}

		require.Equal(t, msg.NetworkServiceEndpoint.Name, expectedForwarderName)

		require.NoError(t, err)
	}
}
