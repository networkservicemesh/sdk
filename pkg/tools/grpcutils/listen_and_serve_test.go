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
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

func TestListenAndServe_NotExistsFolder(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	dir, err := ioutil.TempDir(os.TempDir(), t.Name())
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	socket := path.Join(dir, "folder", "test.sock")
	ctx, cancel := context.WithCancel(context.Background())
	ch := grpcutils.ListenAndServe(ctx, &url.URL{Scheme: "unix", Path: socket}, grpc.NewServer())
	if len(ch) > 0 {
		require.NoError(t, <-ch)
	}
	cancel()
	<-ctx.Done()
}

func TestListenAndServe_ExistSocket(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	dir, err := ioutil.TempDir(os.TempDir(), t.Name())
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	socket := path.Join(dir, "test.sock")
	err = ioutil.WriteFile(socket, []byte("..."), os.ModePerm)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	ch := grpcutils.ListenAndServe(ctx, &url.URL{Scheme: "unix", Path: socket}, grpc.NewServer())
	if len(ch) > 0 {
		require.NoError(t, <-ch)
	}
	cancel()
	<-ctx.Done()
}
