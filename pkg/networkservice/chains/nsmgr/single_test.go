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

package nsmgr_test

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func Test_DNSUsecase(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()

	_, err := domain.Nodes[0].NewEndpoint(ctx, defaultRegistryEndpoint(), sandbox.GenerateTestToken, dnscontext.NewServer(
		&networkservice.DNSConfig{
			DnsServerIps:  []string{"8.8.8.8"},
			SearchDomains: []string{"my.domain1"},
		},
		&networkservice.DNSConfig{
			DnsServerIps:  []string{"8.8.4.4"},
			SearchDomains: []string{"my.domain1"},
		},
	))
	require.NoError(t, err)

	corefilePath := filepath.Join(t.TempDir(), "corefile")
	resolveConfigPath := filepath.Join(t.TempDir(), "resolv.conf")

	err = ioutil.WriteFile(resolveConfigPath, []byte("nameserver 8.8.4.4\nsearch example.com\n"), os.ModePerm)
	require.NoError(t, err)

	const expectedCorefile = ". {\n\tforward . 8.8.4.4\n\tlog\n\treload\n}\nmy.domain1 {\n\tfanout . 8.8.4.4 8.8.8.8\n\tlog\n}"

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, dnscontext.NewClient(
		dnscontext.WithChainContext(ctx),
		dnscontext.WithCorefilePath(corefilePath),
		dnscontext.WithResolveConfigPath(resolveConfigPath),
	))

	conn, err := nsc.Request(ctx, defaultRequest())
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		// #nosec
		b, readFileErr := ioutil.ReadFile(corefilePath)
		if readFileErr != nil {
			return false
		}
		return string(b) == expectedCorefile
	}, time.Second, time.Millisecond*100)

	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	_, err = domain.Nodes[0].EndpointRegistryClient.Unregister(ctx, defaultRegistryEndpoint())
	require.NoError(t, err)
}

func Test_ShouldCorrectlyAddForwardersWithSameNames(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(nil).
		SetRegistryExpiryDuration(sandbox.RegistryExpiryDuration).
		SetContext(ctx).
		Build()

	forwarderReg := &registry.NetworkServiceEndpoint{
		Name: "forwarder",
	}

	nseReg := defaultRegistryEndpoint()

	// 1. Add forwarders
	forwarder1Reg := forwarderReg.Clone()
	_, err := domain.Nodes[0].NewForwarder(ctx, forwarder1Reg, sandbox.GenerateTestToken)
	require.NoError(t, err)

	forwarder2Reg := forwarderReg.Clone()
	_, err = domain.Nodes[0].NewForwarder(ctx, forwarder2Reg, sandbox.GenerateTestToken)
	require.NoError(t, err)

	forwarder3Reg := forwarderReg.Clone()
	_, err = domain.Nodes[0].NewForwarder(ctx, forwarder3Reg, sandbox.GenerateTestToken)
	require.NoError(t, err)

	// 2. Wait for refresh
	<-time.After(sandbox.RegistryExpiryDuration)

	testNSEAndClient(ctx, t, domain, nseReg.Clone())

	// 3. Delete first forwarder
	_, err = domain.Nodes[0].ForwarderRegistryClient.Unregister(ctx, forwarder1Reg)
	require.NoError(t, err)

	testNSEAndClient(ctx, t, domain, nseReg.Clone())

	// 4. Delete last forwarder
	_, err = domain.Nodes[0].ForwarderRegistryClient.Unregister(ctx, forwarder3Reg)
	require.NoError(t, err)

	testNSEAndClient(ctx, t, domain, nseReg.Clone())

	_, err = domain.Nodes[0].ForwarderRegistryClient.Unregister(ctx, forwarder2Reg)
	require.NoError(t, err)
}

func Test_ShouldCorrectlyAddEndpointsWithSameNames(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetRegistryExpiryDuration(sandbox.RegistryExpiryDuration).
		SetContext(ctx).
		Build()

	nseReg := &registry.NetworkServiceEndpoint{
		Name: "endpoint",
	}

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	// 1. Add endpoints
	nse1Reg := nseReg.Clone()
	nse1Reg.NetworkServiceNames = []string{"service-1"}
	_, err := domain.Nodes[0].NewEndpoint(ctx, nse1Reg, sandbox.GenerateTestToken)
	require.NoError(t, err)

	nse2Reg := nseReg.Clone()
	nse2Reg.NetworkServiceNames = []string{"service-2"}
	_, err = domain.Nodes[0].NewEndpoint(ctx, nse2Reg, sandbox.GenerateTestToken)
	require.NoError(t, err)

	// 2. Wait for refresh
	<-time.After(sandbox.RegistryExpiryDuration)

	// 3. Request
	_, err = nsc.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "service-1",
		},
	})
	require.NoError(t, err)

	_, err = nsc.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "service-2",
		},
	})
	require.NoError(t, err)

	// 3. Delete endpoints
	_, err = domain.Nodes[0].EndpointRegistryClient.Unregister(ctx, nse1Reg)
	require.NoError(t, err)

	_, err = domain.Nodes[0].EndpointRegistryClient.Unregister(ctx, nse2Reg)
	require.NoError(t, err)
}

func Test_Local_NoURLUsecase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix sockets are not supported under windows, skipping")
		return
	}
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		UseUnixSockets().
		SetContext(ctx).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetRegistrySupplier(nil).
		Build()

	nseReg := defaultRegistryEndpoint()
	request := defaultRequest()
	counter := &counterServer{}

	_, err := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, counter)
	require.NoError(t, err)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Requests))
	require.Equal(t, 5, len(conn.Path.PathSegments))

	// Simulate refresh from client
	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn2, err := nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn2)
	require.Equal(t, 5, len(conn2.Path.PathSegments))
	require.Equal(t, int32(2), atomic.LoadInt32(&counter.Requests))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&counter.Closes))
}
