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

package client_test

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func TestClientHeal(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	serverUrl := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	serverCancel := startEmptyServer(ctx, t, serverUrl)
	defer serverCancel()

	nsc := client.NewClient(ctx, serverUrl, sandbox.DefaultDialOptions(sandbox.GenerateTestToken))
	_, err := nsc.Request(ctx, &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)

	serverCancel()
	require.Eventually(t, checkURLFree(serverUrl.Host), time.Second, time.Millisecond*10)
	require.NoError(t, ctx.Err())

	// _, err = nsc.Request(ctx, &networkservice.NetworkServiceRequest{})
	// require.Error(t, err)

	serverCancel = startEmptyServer(ctx, t, serverUrl)
	defer serverCancel()

	require.Eventually(t, func() bool {
		_, err = nsc.Request(ctx, &networkservice.NetworkServiceRequest{})
		fmt.Println(err)
		return err == nil
	}, time.Second, time.Millisecond*50)
}

func startEmptyServer(ctx context.Context, t *testing.T, serverUrl *url.URL, ) context.CancelFunc {
	serverCtx, serverCancel := context.WithCancel(ctx)

	nse := endpoint.NewServer(serverCtx, sandbox.GenerateTestToken)

	select {
	case err := <-endpoint.Serve(serverCtx, serverUrl, nse):
		require.NoError(t, err)
	default:
	}

	return serverCancel
}

func checkURLFree(url string) func() bool {
	return func() bool {
		ln, err := net.Listen("tcp", url)
		if err != nil {
			return false
		}
		err = ln.Close()
		return err == nil
	}
}
