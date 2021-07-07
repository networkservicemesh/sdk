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

// +build !windows

package resetting_test

import (
	"context"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/resetting"
)

func Test_ResetTransportCreds_RPC(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), t.Name())
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	var urls = []*url.URL{
		&(url.URL{Scheme: "tcp", Path: "127.0.0.1"}),
		&(url.URL{Scheme: "udp", Path: "127.0.0.1"}),
		&(url.URL{Scheme: "unix", Path: filepath.Join(tmpDir, "listen.on.socket")}),
	}

	for i := 0; i < len(urls); i++ {
		u := urls[i]
		t.Run(t.Name()+", scheme: "+u.Scheme, func(t *testing.T) {
			testResetTransportCredsRPC(t, u)
		})
	}
}

func Test_ResetTransportCreds_Stream(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), t.Name())
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	var urls = []*url.URL{
		&(url.URL{Scheme: "tcp", Path: "127.0.0.1"}),
		&(url.URL{Scheme: "udp", Path: "127.0.0.1"}),
		&(url.URL{Scheme: "unix", Path: filepath.Join(tmpDir, "listen.on.socket")}),
	}

	for i := 0; i < len(urls); i++ {
		u := urls[i]
		t.Run(t.Name()+", scheme: "+u.Scheme, func(t *testing.T) {
			testResetTransportCredsStream(t, u)
		})
	}
}

func testResetTransportCredsStream(t *testing.T, listenOn *url.URL) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	deadline, _ := ctx.Deadline()

	var notifyCh = make(chan struct{})
	defer close(notifyCh)

	creds := resetting.NewCredentials(insecure.NewCredentials(), notifyCh)

	s := grpc.NewServer()

	healthServer := health.NewServer()

	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("test", grpc_health_v1.HealthCheckResponse_SERVING)

	errCh := grpcutils.ListenAndServe(ctx, listenOn, s)
	require.Equal(t, 0, len(errCh))

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(listenOn), grpc.WithTransportCredentials(creds), grpc.WithBlock())
	require.NoError(t, err)
	defer func() {
		_ = cc.Close()
	}()
	c := grpc_health_v1.NewHealthClient(cc)

	stream, err := c.Watch(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "test",
	})
	require.NoError(t, err)

	select {
	case <-ctx.Done():
		t.Error("notifyCh is not read")
		t.FailNow()
	case notifyCh <- struct{}{}:
	}

	require.NoError(t, stream.Context().Err())

	require.Eventually(t, func() bool {
		stream, err = c.Watch(ctx, &grpc_health_v1.HealthCheckRequest{
			Service: "test",
		})
		if err != nil {
			return false
		}
		_, err = stream.Recv()
		return err == nil
	}, time.Until(deadline), time.Millisecond*50)
}

func testResetTransportCredsRPC(t *testing.T, listenOn *url.URL) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	deadline, _ := ctx.Deadline()

	var notifyCh = make(chan struct{})
	defer close(notifyCh)
	creds := resetting.NewCredentials(insecure.NewCredentials(), notifyCh)

	s := grpc.NewServer()

	healthServer := health.NewServer()

	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("test", grpc_health_v1.HealthCheckResponse_SERVING)
	println(listenOn.String())
	errCh := grpcutils.ListenAndServe(ctx, listenOn, s)
	require.Equal(t, 0, len(errCh))

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(listenOn), grpc.WithTransportCredentials(creds), grpc.WithBlock())
	require.NoError(t, err)

	defer func() {
		_ = cc.Close()
	}()

	c := grpc_health_v1.NewHealthClient(cc)

	_, err = c.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "test",
	})
	require.NoError(t, err)

	select {
	case <-ctx.Done():
		t.Error("notifyCh is not read")
		t.FailNow()
	case notifyCh <- struct{}{}:
	}

	require.Eventually(t, func() bool {
		_, err = c.Check(ctx, &grpc_health_v1.HealthCheckRequest{
			Service: "test",
		})
		return err == nil
	}, time.Until(deadline), time.Millisecond*50)
}

type closeCountConn struct {
	net.Conn
	closeCount int32
}

func (c *closeCountConn) Close() error {
	atomic.AddInt32(&c.closeCount, 1)
	return nil
}

func Test_ResettingTransportCreds_ShouldCorrectlyCleanupResources(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	const connLen = 100

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	deadline, _ := ctx.Deadline()

	var notifyCh = make(chan struct{})
	defer close(notifyCh)

	creds := resetting.NewCredentials(insecure.NewCredentials(), notifyCh)

	var inputConns = make([]closeCountConn, connLen)
	var outputConns = make([]net.Conn, 0, connLen)

	for i := 0; i < connLen; i++ {
		conn, _, err := creds.ClientHandshake(ctx, "", &inputConns[i])
		outputConns = append(outputConns, conn)
		require.NoError(t, err)
	}

	for i := 0; i < connLen; i++ {
		require.Equal(t, int32(0), atomic.LoadInt32(&inputConns[i].closeCount))
		_ = outputConns[i].Close()
		require.Equal(t, int32(1), atomic.LoadInt32(&inputConns[i].closeCount))
	}

	notifyCh <- struct{}{}

	for i := 0; i < connLen; i++ {
		require.Equal(t, int32(1), atomic.LoadInt32(&inputConns[i].closeCount))
	}

	for i := 0; i < connLen; i++ {
		_, _, err := creds.ClientHandshake(ctx, "", &inputConns[i])
		require.NoError(t, err)
		require.Equal(t, int32(1), atomic.LoadInt32(&inputConns[i].closeCount))
	}

	notifyCh <- struct{}{}

	for i := 0; i < connLen; i++ {
		id := i

		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&inputConns[id].closeCount) == 2
		}, time.Until(deadline), time.Millisecond, "expected 2 closes: actual: %v. Iteration: %v", atomic.LoadInt32(&inputConns[i].closeCount), i)

		require.Equal(t, int32(2), atomic.LoadInt32(&inputConns[i].closeCount))
	}
}
