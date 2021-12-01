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

// +build linux

package recvfd_test

import (
	"context"
	"net"
	"net/url"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/edwarnicke/grpcfd"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"

	registryserver "github.com/networkservicemesh/sdk/pkg/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	registryrecvfd "github.com/networkservicemesh/sdk/pkg/registry/common/recvfd"
	"github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	registryserialize "github.com/networkservicemesh/sdk/pkg/registry/common/serialize"
	registrychain "github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type checkNseRecvfdServer struct {
	onRecvFile func()

	t *testing.T
}

type notifiableFDTransceiver struct {
	grpcfd.FDTransceiver
	net.Addr

	onRecvFile func()
}

func (n *checkNseRecvfdServer) Unregister(ctx context.Context, endpoint *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, endpoint)
}

func (n *checkNseRecvfdServer) Register(ctx context.Context, endpoint *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	p, ok := peer.FromContext(ctx)
	require.True(n.t, ok)

	transceiver, ok := p.Addr.(grpcfd.FDTransceiver)
	require.True(n.t, ok)

	p.Addr = &notifiableFDTransceiver{
		FDTransceiver: transceiver,
		Addr:          p.Addr,
		onRecvFile:    n.onRecvFile,
	}

	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, endpoint)
}

func (n *checkNseRecvfdServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (w *notifiableFDTransceiver) RecvFileByURL(urlStr string) (<-chan *os.File, error) {
	recv, err := w.FDTransceiver.RecvFileByURL(urlStr)
	if err != nil {
		return nil, err
	}

	var fileCh = make(chan *os.File)
	go func() {
		for f := range recv {
			runtime.SetFinalizer(f, func(file *os.File) {
				w.onRecvFile()
			})
			fileCh <- f
		}
	}()

	return fileCh, nil
}

func startServer(ctx context.Context, t *testing.T, testRegistry registryserver.Registry, serveURL *url.URL) {
	var grpcServer = grpc.NewServer(grpc.Creds(grpcfd.TransportCredentials(insecure.NewCredentials())))

	testRegistry.Register(grpcServer)

	var errCh = grpcutils.ListenAndServe(ctx, serveURL, grpcServer)
	require.Len(t, errCh, 0)
}

func TestNseRecvfdServerClosesFile(t *testing.T) {
	var ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var dir = t.TempDir()

	s, err := os.Create(path.Join(dir, "test.sock"))
	require.NoError(t, err)

	err = s.Close()
	require.NoError(t, err)

	fileClosedContext, cancelFunc := context.WithCancel(context.Background())

	var nsRegistry = registrychain.NewNetworkServiceRegistryServer(
		registryserialize.NewNetworkServiceRegistryServer(),
		memory.NewNetworkServiceRegistryServer(),
	)

	var nseRegistry = registrychain.NewNetworkServiceEndpointRegistryServer(
		registryserialize.NewNetworkServiceEndpointRegistryServer(),
		&checkNseRecvfdServer{
			t:          t,
			onRecvFile: cancelFunc,
		},
		registryrecvfd.NewNetworkServiceEndpointRegistryServer(),
		memory.NewNetworkServiceEndpointRegistryServer(),
	)

	var regURL = &url.URL{Scheme: "unix", Path: s.Name()}

	var dialOptions = []grpc.DialOption{
		grpc.WithTransportCredentials(
			grpcfd.TransportCredentials(insecure.NewCredentials()),
		),
		grpc.WithDefaultCallOptions(
			grpc.PerRPCCredentials(token.NewPerRPCCredentials(sandbox.GenerateTestToken)),
		),
		grpcfd.WithChainStreamInterceptor(),
		grpcfd.WithChainUnaryInterceptor(),
		sandbox.WithInsecureRPCCredentials(),
		sandbox.WithInsecureStreamRPCCredentials(),
	}

	var nseClient = registrychain.NewNetworkServiceEndpointRegistryClient(
		registryserialize.NewNetworkServiceEndpointRegistryClient(),
		refresh.NewNetworkServiceEndpointRegistryClient(ctx),
		connect.NewNetworkServiceEndpointRegistryClient(ctx, regURL,
			connect.WithNSEAdditionalFunctionality(
				sendfd.NewNetworkServiceEndpointRegistryClient()),
			connect.WithDialOptions(dialOptions...),
		))

	startServer(ctx, t, registryserver.NewServer(nsRegistry, nseRegistry), regURL)
	require.NoError(t, err)

	var testEndpoint = &registry.NetworkServiceEndpoint{
		Name:                "test-endpoint",
		NetworkServiceNames: []string{"test"},
		Url:                 regURL.String(),
	}

	_, err = nseClient.Register(ctx, testEndpoint.Clone())
	require.NoError(t, err)

	_, err = nseClient.Unregister(ctx, testEndpoint.Clone())
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		runtime.GC()
		return fileClosedContext.Err() != nil
	}, time.Second, time.Millisecond*100)
}
