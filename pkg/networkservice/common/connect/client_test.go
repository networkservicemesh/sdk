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

package connect

import (
	"context"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	parallelCount = 100
)

func TestConnectClient_DialTimeout(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	dir, err := os.MkdirTemp(os.TempDir(), t.Name())
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	socket := path.Join(dir, "test.sock")

	lis, err := net.Listen("unix", socket)
	require.NoError(t, err)
	defer lis.Close()

	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		),
	}

	conn, _ := grpc.Dial("unix://"+socket, dialOptions...)

	ready := waitForReady(ctx, conn, time.Second*2)
	require.False(t, ready)
	err = conn.Close()
	require.NoError(t, err)
}
