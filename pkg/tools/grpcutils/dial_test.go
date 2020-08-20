// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package grpcutils_test

import (
	"context"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

func TestDialContextTimeout(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	s := grpc.NewServer()
	dir, err := ioutil.TempDir(os.TempDir(), t.Name())
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	u := &url.URL{Scheme: "unix", Path: path.Join(dir, "server.sock")}
	l, err := net.Listen(u.Scheme, u.Path)
	defer func() {
		_ = l.Close()
	}()
	require.NoError(t, err)

	go func() {
		<-time.After(time.Millisecond * 200)
		_ = s.Serve(l)
	}()

	conn, err := grpcutils.DialContext(context.Background(), u.String(), grpc.WithInsecure())
	require.NoError(t, err)
	require.NotNil(t, conn)
	_ = conn.Close()
}
