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

	"github.com/stretchr/testify/assert"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type contextKeyType string

const testDataKey contextKeyType = "testData"

type TestData struct {
	clientURL *url.URL
}

func withTestData(parent context.Context, testData *TestData) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, testDataKey, testData)
}

func testData(ctx context.Context) *TestData {
	if rv, ok := ctx.Value(testDataKey).(*TestData); ok {
		return rv
	}
	return nil
}

type testNetworkServiceServer struct{}

func (c *testNetworkServiceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	testData(ctx).clientURL = clienturl.ClientURL(ctx)
	return next.Server(ctx).Request(ctx, request)
}

func (c *testNetworkServiceServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	testData(ctx).clientURL = clienturl.ClientURL(ctx)
	return next.Server(ctx).Close(ctx, conn)
}

func TestAddURLInEmptyContext(t *testing.T) {
	clientURL := &url.URL{
		Scheme: "ipv4",
		Path:   "192.168.0.1",
	}
	client := next.NewNetworkServiceServer(clienturl.NewServer(clientURL), &testNetworkServiceServer{})

	ctx := withTestData(context.Background(), &TestData{})
	_, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	assert.Nil(t, err)
	assert.Equal(t, clientURL, testData(ctx).clientURL)

	ctx = withTestData(context.Background(), &TestData{})
	_, err = client.Close(ctx, &networkservice.Connection{})
	assert.Nil(t, err)
	assert.Equal(t, clientURL, testData(ctx).clientURL)
}

func TestOverwriteURL(t *testing.T) {
	clientURL := &url.URL{
		Scheme: "ipv4",
		Path:   "192.168.0.1",
	}
	previousURL := &url.URL{
		Scheme: "unix",
		Path:   "/var/run/nse-1.sock",
	}
	client := next.NewNetworkServiceServer(clienturl.NewServer(clientURL), &testNetworkServiceServer{})

	ctx := withTestData(context.Background(), &TestData{})
	ctx = clienturl.WithClientURL(ctx, previousURL)
	_, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	assert.Nil(t, err)
	assert.Equal(t, clientURL, testData(ctx).clientURL)

	ctx = withTestData(context.Background(), &TestData{})
	ctx = clienturl.WithClientURL(ctx, previousURL)
	_, err = client.Close(ctx, &networkservice.Connection{})
	assert.Nil(t, err)
	assert.Equal(t, clientURL, testData(ctx).clientURL)
}

func TestOverwriteURLByNil(t *testing.T) {
	var clientURL *url.URL = nil
	previousURL := &url.URL{
		Scheme: "unix",
		Path:   "/var/run/nse-1.sock",
	}
	client := next.NewNetworkServiceServer(clienturl.NewServer(clientURL), &testNetworkServiceServer{})

	ctx := withTestData(context.Background(), &TestData{})
	ctx = clienturl.WithClientURL(ctx, previousURL)
	_, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	assert.Nil(t, err)
	assert.Equal(t, clientURL, testData(ctx).clientURL)

	ctx = withTestData(context.Background(), &TestData{})
	ctx = clienturl.WithClientURL(ctx, previousURL)
	_, err = client.Close(ctx, &networkservice.Connection{})
	assert.Nil(t, err)
	assert.Equal(t, clientURL, testData(ctx).clientURL)
}
