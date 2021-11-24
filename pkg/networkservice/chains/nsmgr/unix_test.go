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

//+build !windows

package nsmgr_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	linuxOsName = "linux"
)

func Test_Local_NoURLUsecase(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		UseUnixSockets().
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)
	request := defaultRequest(nsReg.Name)
	counter := new(count.Server)

	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.Requests())
	require.Equal(t, 4, len(conn.Path.PathSegments))

	// Simulate refresh from client
	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn2, err := nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn2)
	require.Equal(t, 4, len(conn2.Path.PathSegments))
	require.Equal(t, 2, counter.Requests())

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, 1, counter.Closes())
}

func Test_MultiForwarderSendfd(t *testing.T) {
	if runtime.GOOS != linuxOsName {
		t.Skip("sendfd works only on linux")
	}
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	errorServer := injecterror.NewServer(
		injecterror.WithRequestErrorTimes(0),
		injecterror.WithCloseErrorTimes(),
	)
	domain := sandbox.NewBuilder(ctx, t).
		UseUnixSockets().
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, _ int) {
			node.NewNSMgr(ctx, "nsmgr", nil, sandbox.GenerateTestToken, nsmgr.NewServer)
			node.NewForwarder(ctx, &registry.NetworkServiceEndpoint{
				Name:                "forwarder-1",
				NetworkServiceNames: []string{"forwarder"},
				NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
					"forwarder": {
						Labels: map[string]string{
							"p2p": "true",
						},
					},
				},
			}, sandbox.GenerateTestToken, errorServer, recvfd.NewServer())
			node.NewForwarder(ctx, &registry.NetworkServiceEndpoint{
				Name:                "forwarder-2",
				NetworkServiceNames: []string{"forwarder"},
				NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
					"forwarder": {
						Labels: map[string]string{
							"p2p": "true",
						},
					},
				},
			}, sandbox.GenerateTestToken, errorServer, recvfd.NewServer())
		}).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)
	counter := new(count.Server)

	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, kernel.NewClient(), sendfd.NewClient())

	request := defaultRequest(nsReg.Name)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.Requests())
	require.Equal(t, 4, len(conn.Path.PathSegments))

	// Simulate refresh from client
	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn2, err := nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn2)
	require.Equal(t, 4, len(conn2.Path.PathSegments))
	require.Equal(t, 2, counter.Requests())

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, 1, counter.Closes())
}

func Test_TimeoutRecvfd(t *testing.T) {
	if runtime.GOOS != linuxOsName {
		t.Skip("recvfd works only on linux")
	}
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		UseUnixSockets().
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, _ int) {
			node.NewNSMgr(ctx, "nsmgr", nil, sandbox.GenerateTestToken, nsmgr.NewServer)
			node.NewForwarder(ctx, &registry.NetworkServiceEndpoint{
				Name:                "forwarder-1",
				NetworkServiceNames: []string{"forwarder"},
				NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
					"forwarder": {
						Labels: map[string]string{
							"p2p": "true",
						},
					},
				},
			}, sandbox.GenerateTestToken, recvfd.NewServer())
		}).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService())
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateExpiringToken(0), kernel.NewClient(), sendfd.NewClient())

	request := defaultRequest(nsReg.Name)

	conn, err := nsc.Request(ctx, request.Clone())
	require.Nil(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 4, len(conn.Path.PathSegments))
}
