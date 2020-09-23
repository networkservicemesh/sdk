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

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
)

func TestAddURLInEmptyContext(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	clientURL := &url.URL{
		Scheme: "ipv4",
		Path:   "192.168.0.1",
	}
	client := next.NewNetworkServiceServer(clienturl.NewServer(clientURL), checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
		assert.Equal(t, clientURL, clienturlctx.ClientURL(ctx))
	}))

	_, err := client.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	assert.Nil(t, err)
	_, err = client.Close(context.Background(), &networkservice.Connection{})
	assert.Nil(t, err)
}

func TestOverwriteURL(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	clientURL := &url.URL{
		Scheme: "ipv4",
		Path:   "192.168.0.1",
	}
	previousURL := &url.URL{
		Scheme: "unix",
		Path:   "/var/run/nse-1.sock",
	}
	client := next.NewNetworkServiceServer(clienturl.NewServer(clientURL), checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
		assert.Equal(t, clientURL, clienturlctx.ClientURL(ctx))
	}))

	ctx := clienturlctx.WithClientURL(context.Background(), previousURL)
	_, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	assert.Nil(t, err)

	ctx = clienturlctx.WithClientURL(context.Background(), previousURL)
	_, err = client.Close(ctx, &networkservice.Connection{})
	assert.Nil(t, err)
}

func TestOverwriteURLByNil(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	var clientURL *url.URL = nil
	previousURL := &url.URL{
		Scheme: "unix",
		Path:   "/var/run/nse-1.sock",
	}
	client := next.NewNetworkServiceServer(clienturl.NewServer(clientURL), checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
		assert.Equal(t, clientURL, clienturlctx.ClientURL(ctx))
	}))

	ctx := clienturlctx.WithClientURL(context.Background(), previousURL)
	_, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	assert.Nil(t, err)

	ctx = clienturlctx.WithClientURL(context.Background(), previousURL)
	_, err = client.Close(ctx, &networkservice.Connection{})
	assert.Nil(t, err)
}
