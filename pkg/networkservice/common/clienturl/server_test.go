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

// Package clienturl_test provides a tests for package 'clienturl'
package clienturl_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type testNetworkServiceServer struct {
	t    *testing.T
	want *url.URL
}

func (c *testNetworkServiceServer) Request(ctx context.Context, in *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	assert.Equal(c.t, c.want, clienturl.ClientURL(ctx))
	return in.GetConnection(), nil
}

func (c *testNetworkServiceServer) Close(ctx context.Context, _ *networkservice.Connection) (*empty.Empty, error) {
	assert.Equal(c.t, c.want, clienturl.ClientURL(ctx))
	return &empty.Empty{}, nil
}

var testData = []struct {
	name  string
	ctx   context.Context
	given *url.URL
	want  *url.URL
}{
	{
		"add client url in empty context",
		context.Background(),
		&url.URL{
			Scheme: "ipv4",
			Path:   "192.168.0.1",
		},
		&url.URL{
			Scheme: "ipv4",
			Path:   "192.168.0.1",
		},
	},
	{
		"overwrite client url",
		clienturl.WithClientURL(context.Background(), &url.URL{
			Scheme: "unix",
			Path:   "/var/run/nse-1.sock",
		}),
		&url.URL{
			Scheme: "ipv4",
			Path:   "192.168.0.1",
		},
		&url.URL{
			Scheme: "ipv4",
			Path:   "192.168.0.1",
		},
	},
	{
		"overwrite client url by nil",
		clienturl.WithClientURL(context.Background(), &url.URL{
			Scheme: "unix",
			Path:   "/var/run/nse-1.sock",
		}),
		nil,
		nil,
	},
}

func Test_clientUrlServer(t *testing.T) {
	for _, data := range testData {
		test := data
		t.Run(test.name, func(t *testing.T) {
			testServer(test.ctx, test.given, test.want, t)
		})
	}
}

func testServer(ctx context.Context, given, want *url.URL, t *testing.T) {
	client := next.NewNetworkServiceServer(clienturl.NewServer(given), &testNetworkServiceServer{t: t, want: want})
	_, _ = client.Request(ctx, &networkservice.NetworkServiceRequest{})
	_, _ = client.Close(ctx, &networkservice.Connection{})
}
