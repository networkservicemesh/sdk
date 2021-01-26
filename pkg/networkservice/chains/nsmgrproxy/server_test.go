// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package nsmgrproxy_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func TestNSMGR_InterdomainUseCase(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	const remoteRegistryDomain = "domain2.local.registry"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	dnsServer := new(sandbox.FakeDNSResolver)

	domain1 := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetContext(ctx).
		SetDNSResolver(dnsServer).
		Build()
	defer domain1.Cleanup()

	domain2 := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetDNSResolver(dnsServer).
		SetContext(ctx).
		Build()
	defer domain2.Cleanup()

	require.NoError(t, dnsServer.Register(remoteRegistryDomain, domain2.Registry.URL))

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{"my-service-interdomain"},
	}

	_, err := domain2.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)
	require.NoError(t, err)

	nsc := domain1.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service-interdomain@" + remoteRegistryDomain,
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 9, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 9, len(conn.Path.PathSegments))
}
