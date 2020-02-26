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

// Package filtermechanisms_test provides a tests for package 'filtermechanisms'
package filtermechanisms_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/srv6"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/filtermechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type contextKeyType string

const testDataKey contextKeyType = "testData"

type TestData struct {
	mechanisms []*networkservice.Mechanism
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

type testNetworkServiceClient struct{}

func (s testNetworkServiceClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	testData(ctx).mechanisms = request.GetMechanismPreferences()
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (s testNetworkServiceClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func request() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Cls:  cls.LOCAL,
				Type: memif.MECHANISM,
			},
			{
				Cls:  cls.LOCAL,
				Type: kernel.MECHANISM,
			},
			{
				Cls:  cls.REMOTE,
				Type: srv6.MECHANISM,
			},
			{
				Cls:  cls.REMOTE,
				Type: vxlan.MECHANISM,
			},
			{
				Cls:  "NOT_A_CLS",
				Type: "NOT_A_TYPE",
			},
		},
	}
}

func TestNewClient_FilterUnixType(t *testing.T) {
	client := next.NewNetworkServiceClient(filtermechanisms.NewClient(), &testNetworkServiceClient{})
	ctx := withTestData(context.Background(), &TestData{})
	ctx = clienturl.WithClientURL(ctx, &url.URL{
		Scheme: "unix",
		Path:   "/var/run/nse-1.sock",
	})
	_, err := client.Request(ctx, request())
	assert.Nil(t, err)
	expected := []*networkservice.Mechanism{
		{
			Cls:  cls.LOCAL,
			Type: memif.MECHANISM,
		},
		{
			Cls:  cls.LOCAL,
			Type: kernel.MECHANISM,
		},
	}
	assert.Equal(t, expected, testData(ctx).mechanisms)
}

func TestNewClient_FilterNonUnixType(t *testing.T) {
	client := next.NewNetworkServiceClient(filtermechanisms.NewClient(), &testNetworkServiceClient{})
	ctx := withTestData(context.Background(), &TestData{})
	ctx = clienturl.WithClientURL(ctx, &url.URL{
		Scheme: "ipv4",
		Path:   "192.168.0.1",
	})
	_, err := client.Request(ctx, request())
	assert.Nil(t, err)
	expected := []*networkservice.Mechanism{
		{
			Cls:  cls.REMOTE,
			Type: srv6.MECHANISM,
		},
		{
			Cls:  cls.REMOTE,
			Type: vxlan.MECHANISM,
		},
	}
	assert.Equal(t, expected, testData(ctx).mechanisms)
}
