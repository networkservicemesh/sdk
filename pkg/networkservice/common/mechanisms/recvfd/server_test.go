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
	"fmt"
	"net/url"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/edwarnicke/grpcfd"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcfdutils"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func createFile(fileName string, t *testing.T) (inodeURLStr string, fileClosedContext context.Context, cancelFunc func()) {
	f, err := os.Create(fileName)
	require.NoError(t, err, "Failed to create and open a file: %v", err)

	err = f.Close()
	require.NoError(t, err, "Failed to close file: %v", err)

	fileClosedContext, cancelFunc = context.WithCancel(context.Background())

	inodeURL, err := grpcfd.FilenameToURL(fileName)
	require.NoError(t, err)

	return inodeURL.String(), fileClosedContext, cancelFunc
}

func startServer(ctx context.Context, t *testing.T, testServerChain *networkservice.NetworkServiceServer, serveURL *url.URL) {
	grpcServer := grpc.NewServer(grpc.Creds(grpcfd.TransportCredentials(insecure.NewCredentials())))
	networkservice.RegisterNetworkServiceServer(grpcServer, *testServerChain)

	errCh := grpcutils.ListenAndServe(ctx, serveURL, grpcServer)

	require.Len(t, errCh, 0)
}

func createClient(ctx context.Context, u *url.URL) networkservice.NetworkServiceClient {
	return client.NewClient(
		ctx,
		client.WithClientURL(sandbox.CloneURL(u)),
		client.WithDialOptions(grpc.WithTransportCredentials(
			grpcfd.TransportCredentials(insecure.NewCredentials())),
		),
		client.WithDialTimeout(time.Second),
		client.WithoutRefresh(),
		client.WithAdditionalFunctionality(sendfd.NewClient()))
}

func TestRecvfdClosesSingleFile(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	dir := t.TempDir()

	s, err := os.Create(path.Join(dir, "test.sock"))
	require.NoError(t, err)

	testFileName := path.Join(dir, t.Name()+".test")

	inodeURLStr, fileClosedContext, cancelFunc := createFile(testFileName, t)

	onFileClosedCallbacks := map[string]func(){
		inodeURLStr: cancelFunc,
	}

	serveURL := &url.URL{Scheme: "unix", Path: s.Name()}

	testChain := chain.NewNetworkServiceServer(
		checkcontext.NewServer(t, func(t *testing.T, c context.Context) {
			injectErr := grpcfdutils.InjectOnFileReceivedCallback(c, func(fileName string, file *os.File) {
				runtime.SetFinalizer(file, func(file *os.File) {
					onFileClosedCallback, ok := onFileClosedCallbacks[fileName]
					if ok {
						onFileClosedCallback()
					}
				})
			})

			require.NoError(t, injectErr)
		}),
		recvfd.NewServer())

	startServer(ctx, t, &testChain, serveURL)
	testClient := createClient(ctx, serveURL)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Cls:  cls.LOCAL,
				Type: kernel.MECHANISM,
				Parameters: map[string]string{
					common.InodeURL: "file:" + testFileName,
				},
			},
		},
	}

	conn, err := testClient.Request(ctx, request)
	require.NoError(t, err)

	_, err = testClient.Close(ctx, conn)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		runtime.GC()
		return fileClosedContext.Err() != nil
	}, time.Second, time.Millisecond*100)
}

func TestRecvfdClosesMultipleFiles(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	dir := t.TempDir()

	s, err := os.Create(path.Join(dir, "test.sock"))
	require.NoError(t, err)

	const numFiles = 3
	fileClosedContexts := make([]context.Context, numFiles)
	onFileClosedCallbacks := make(map[string]func(), numFiles)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: make([]*networkservice.Mechanism, numFiles),
	}

	var filePath string
	for i := 0; i < numFiles; i++ {
		filePath = path.Join(dir, fmt.Sprintf("%s.test%d", t.Name(), i))

		inodeURLStr, fileClosedContext, cancelFunc := createFile(filePath, t)
		onFileClosedCallbacks[inodeURLStr] = cancelFunc
		fileClosedContexts[i] = fileClosedContext

		request.MechanismPreferences = append(request.MechanismPreferences,
			&networkservice.Mechanism{
				Cls:  cls.LOCAL,
				Type: kernel.MECHANISM,
				Parameters: map[string]string{
					common.InodeURL: "file:" + filePath,
				},
			})
	}

	serveURL := &url.URL{Scheme: "unix", Path: s.Name(), Host: "0.0.0.0:5000"}

	testChain := chain.NewNetworkServiceServer(
		checkcontext.NewServer(t, func(t *testing.T, c context.Context) {
			e := grpcfdutils.InjectOnFileReceivedCallback(c, func(inodeURLStr string, file *os.File) {
				runtime.SetFinalizer(file, func(file *os.File) {
					onFileClosedCallback, ok := onFileClosedCallbacks[inodeURLStr]
					if ok {
						onFileClosedCallback()
					}
				})
			})

			require.NoError(t, e)
		}),
		recvfd.NewServer())

	startServer(ctx, t, &testChain, serveURL)
	testClient := createClient(ctx, serveURL)

	conn, err := testClient.Request(ctx, request)
	require.NoError(t, err)

	_, err = testClient.Close(ctx, conn)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		runtime.GC()
		return fileClosedContexts[0].Err() != nil
	}, time.Second, time.Millisecond*100)

	require.Eventually(t, func() bool {
		runtime.GC()
		return fileClosedContexts[1].Err() != nil
	}, time.Second, time.Millisecond*100)

	require.Eventually(t, func() bool {
		runtime.GC()
		return fileClosedContexts[2].Err() != nil
	}, time.Second, time.Millisecond*100)
}
