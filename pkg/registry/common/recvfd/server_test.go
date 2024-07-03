// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2024 Cisco and/or its affiliates.
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

//go:build linux
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
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"

	registryserver "github.com/networkservicemesh/sdk/pkg/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dial"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	registryrecvfd "github.com/networkservicemesh/sdk/pkg/registry/common/recvfd"
	"github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcfdutils"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

func getFileInfo(fileName string, t *testing.T) (inodeURLStr string, fileClosedContext context.Context, cancelFunc func()) {
	fileClosedContext, cancelFunc = context.WithCancel(context.Background())

	inodeURL, err := grpcfd.FilenameToURL(fileName)
	require.NoError(t, err)

	return inodeURL.String(), fileClosedContext, cancelFunc
}

func startServer(ctx context.Context, t *testing.T, testRegistry registryserver.Registry, serveURL *url.URL) {
	var grpcServer = grpc.NewServer(grpc.Creds(grpcfd.TransportCredentials(insecure.NewCredentials())))

	testRegistry.Register(grpcServer)

	var errCh = grpcutils.ListenAndServe(ctx, serveURL, grpcServer)
	require.Len(t, errCh, 0)
}

func TestNseRecvfdServerClosesFile(t *testing.T) {
	var ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var nsRegistry = chain.NewNetworkServiceRegistryServer(
		begin.NewNetworkServiceRegistryServer(),
		memory.NewNetworkServiceRegistryServer(),
	)

	var onFileClosedCallbacks = make(map[string]func())

	var nseRegistry = chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		checkcontext.NewNSEServer(t, func(t *testing.T, c context.Context) {
			err := grpcfdutils.InjectOnFileReceivedCallback(c, func(inodeURLStr string, file *os.File) {
				runtime.SetFinalizer(file, func(file *os.File) {
					onFileClosedCallback, ok := onFileClosedCallbacks[inodeURLStr]
					if ok {
						onFileClosedCallback()
					}
				})
			})

			require.NoError(t, err)
		}),
		registryrecvfd.NewNetworkServiceEndpointRegistryServer(),
		memory.NewNetworkServiceEndpointRegistryServer(),
	)

	var dir = t.TempDir()
	var testFileName = path.Join(dir, t.Name()+".sock")
	var regURL = &url.URL{Scheme: "unix", Path: testFileName}

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

	var nseClient = chain.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		refresh.NewNetworkServiceEndpointRegistryClient(ctx),

		chain.NewNetworkServiceEndpointRegistryClient(
			clienturl.NewNetworkServiceEndpointRegistryClient(regURL),
			clientconn.NewNetworkServiceEndpointRegistryClient(),
			dial.NewNetworkServiceEndpointRegistryClient(ctx,
				dial.WithDialOptions(dialOptions...),
				dial.WithDialTimeout(time.Second),
			),
			sendfd.NewNetworkServiceEndpointRegistryClient(),
			connect.NewNetworkServiceEndpointRegistryClient(),
		),
	)

	startServer(ctx, t, registryserver.NewServer(nsRegistry, nseRegistry), regURL)

	// setting onRecvFile after starting the server as we're re-creating a socket in grpcutils.ListenAndServe
	// in registry case we're passing a socket
	inodeURLStr, fileClosedContext, cancelFunc := getFileInfo(testFileName, t)
	onFileClosedCallbacks[inodeURLStr] = cancelFunc

	var testEndpoint = &registry.NetworkServiceEndpoint{
		Name:                "test-endpoint",
		NetworkServiceNames: []string{"test"},
		Url:                 regURL.String(),
	}

	_, err := nseClient.Register(ctx, testEndpoint.Clone())
	require.NoError(t, err)

	_, err = nseClient.Unregister(ctx, testEndpoint.Clone())
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		runtime.GC()
		return fileClosedContext.Err() != nil
	}, time.Second, time.Millisecond*100)
}

type myFDTransceiver struct {
	net.Addr
	ch chan *os.File
}

func (f *myFDTransceiver) RecvFD(dev, inode uint64) <-chan uintptr {
	return nil
}
func (f *myFDTransceiver) RecvFile(dev, ino uint64) <-chan *os.File {
	return nil
}
func (f *myFDTransceiver) RecvFileByURL(urlStr string) (<-chan *os.File, error) {
	return f.ch, nil
}

func (f *myFDTransceiver) SendFD(fd uintptr) <-chan error {
	return nil
}
func (f *myFDTransceiver) SendFile(file grpcfd.SyscallConn) <-chan error {
	return nil
}
func (f *myFDTransceiver) SendFilename(filename string) <-chan error {
	return nil
}

var _ grpcfd.FDTransceiver = (*myFDTransceiver)(nil)

func (f *myFDTransceiver) RecvFDByURL(urlStr string) (<-chan uintptr, error) {
	return nil, nil
}

func TestNseRecvfdServer_ClosesFile(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	f := myFDTransceiver{ch: make(chan *os.File)}
	p := peer.Peer{
		Addr: &f,
	}
	s := registryrecvfd.NewNetworkServiceEndpointRegistryServer()

	defer cancel()

	go func() {
		time.Sleep(time.Second / 2)
		close(f.ch)
	}()

	ctx = peer.NewContext(ctx, &p)

	_, _ = s.Register(ctx, &registry.NetworkServiceEndpoint{Name: "test", Url: "inode://proc/1/fd/1"})
	_, _ = s.Unregister(ctx, &registry.NetworkServiceEndpoint{Name: "test", Url: "inode://proc/1/fd/1"})
}
