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

package clienturl_test

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
)

const (
	dialTimeout = 100 * time.Millisecond
)

func TestClientURLClient_Dial(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	listener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	go func() {
		_ = grpc.NewServer().Serve(listener)
	}()

	u, err := url.Parse("tcp://" + listener.Addr().String())
	require.NoError(t, err)

	client := clienturl.NewClient(clienturlctx.WithClientURL(ctx, u), dialTimeout, func(context.Context, grpc.ClientConnInterface) networkservice.NetworkServiceClient {
		return null.NewClient()
	}, grpc.WithInsecure())

	_, err = client.Request(ctx, new(networkservice.NetworkServiceRequest))
	require.NoError(t, err)
}

func TestClientURLClient_DialTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	listener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	u, err := url.Parse("tcp://" + listener.Addr().String())
	require.NoError(t, err)

	client := clienturl.NewClient(clienturlctx.WithClientURL(ctx, u), dialTimeout, func(context.Context, grpc.ClientConnInterface) networkservice.NetworkServiceClient {
		return null.NewClient()
	}, grpc.WithInsecure())

	timer := time.AfterFunc(time.Second/2, t.FailNow)

	_, err = client.Request(ctx, new(networkservice.NetworkServiceRequest))
	require.Error(t, err)

	timer.Stop()
}
