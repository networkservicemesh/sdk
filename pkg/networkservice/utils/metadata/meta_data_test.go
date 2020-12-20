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

package metadata_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

const (
	testKey = "test"
)

func testServer(server networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return next.NewNetworkServiceServer(
		updatepath.NewServer("server"),
		server,
	)
}

func testRequest(conn *networkservice.Connection) *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: conn,
	}
}

type sample struct {
	name string
	test func(t *testing.T, server networkservice.NetworkServiceServer, isClient bool)
}

var samples = []*sample{
	{
		name: "Request",
		test: func(t *testing.T, server networkservice.NetworkServiceServer, isClient bool) {
			var actual, expected map[string]string = nil, map[string]string{"a": "A"}

			chainServer := next.NewNetworkServiceServer(
				testServer(server),
				checkcontext.NewServer(t, func(_ *testing.T, ctx context.Context) {
					metadata.Map(ctx, isClient).Store(testKey, expected)
				}),
			)
			conn, err := chainServer.Request(context.TODO(), testRequest(nil))
			require.NoError(t, err)

			chainServer = next.NewNetworkServiceServer(
				testServer(server),
				checkcontext.NewServer(t, func(_ *testing.T, ctx context.Context) {
					if raw, ok := metadata.Map(ctx, isClient).Load(testKey); ok {
						actual = raw.(map[string]string)
					} else {
						actual = nil
					}
				}),
			)
			_, err = chainServer.Request(context.TODO(), testRequest(conn))
			require.NoError(t, err)

			require.Equal(t, expected, actual)
		},
	},
	{
		name: "Request failed",
		test: func(t *testing.T, server networkservice.NetworkServiceServer, _ bool) {
			chainServer := next.NewNetworkServiceServer(
				testServer(server),
				injecterror.NewServer(errors.New("error")),
			)
			_, err := chainServer.Request(context.TODO(), testRequest(nil))
			require.Error(t, err)
		},
	},
	{
		name: "Close",
		test: func(t *testing.T, server networkservice.NetworkServiceServer, isClient bool) {
			data := map[string]string{"a": "A"}

			chainServer := next.NewNetworkServiceServer(
				testServer(server),
				checkcontext.NewServer(t, func(_ *testing.T, ctx context.Context) {
					metadata.Map(ctx, isClient).Store(testKey, data)
				}),
			)
			conn, err := chainServer.Request(context.TODO(), testRequest(nil))
			require.NoError(t, err)

			_, err = testServer(server).Close(context.TODO(), conn)
			require.NoError(t, err)

			chainServer = next.NewNetworkServiceServer(
				testServer(server),
				checkcontext.NewServer(t, func(_ *testing.T, ctx context.Context) {
					if raw, ok := metadata.Map(ctx, isClient).Load(testKey); ok {
						data = raw.(map[string]string)
					} else {
						data = nil
					}
				}),
			)
			_, err = chainServer.Request(context.TODO(), testRequest(conn))
			require.NoError(t, err)

			require.Nil(t, data)
		},
	},
}

func TestMetaDataServer(t *testing.T) {
	for i := range samples {
		sample := samples[i]
		t.Run(sample.name, func(t *testing.T) {
			sample.test(t, metadata.NewServer(), false)
		})
	}
}

func TestMetaDataClient(t *testing.T) {
	for i := range samples {
		sample := samples[i]
		t.Run(sample.name, func(t *testing.T) {
			sample.test(t, adapters.NewClientToServer(metadata.NewClient()), true)
		})
	}
}
