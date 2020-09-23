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

// Package localbypass_test contains tests for package 'localbypass'
package localbypass_test

import (
	"context"
	"net"
	"net/url"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/networkservicemesh/sdk/pkg/registry/common/seturl"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/localbypass"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type contextKeyType string

const (
	testDataKey contextKeyType = "testData"
)

func withTestData(parent context.Context, testData *TestData) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, testDataKey, testData)
}

func testData(ctx context.Context) *TestData {
	if rv, ok := ctx.Value(testDataKey).(*TestData); ok {
		return rv
	}
	return nil
}

type TestData struct {
	clientURL *url.URL
}

type testNetworkServiceServer struct {
}

func (s testNetworkServiceServer) Request(ctx context.Context, in *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	testData(ctx).clientURL = clienturlctx.ClientURL(ctx)
	return in.GetConnection(), nil
}

func (s testNetworkServiceServer) Close(ctx context.Context, _ *networkservice.Connection) (*empty.Empty, error) {
	testData(ctx).clientURL = clienturlctx.ClientURL(ctx)
	return &empty.Empty{}, nil
}

func TestNewServer_NSENotPresented(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	var localBypassRegistryServer registry.NetworkServiceEndpointRegistryServer
	localBypassNetworkServiceServer := localbypass.NewServer(&localBypassRegistryServer)
	server := next.NewNetworkServiceServer(localBypassNetworkServiceServer, &testNetworkServiceServer{})
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkServiceEndpointName: "nse-1",
		},
	}

	ctx := withTestData(context.Background(), &TestData{})
	_, err := server.Request(ctx, request)
	assert.Nil(t, err)
	assert.Nil(t, testData(ctx).clientURL)

	ctx = withTestData(context.Background(), &TestData{})
	_, err = server.Close(ctx, request.GetConnection())
	assert.Nil(t, err)
	assert.Nil(t, testData(ctx).clientURL)
}

func TestNewServer_UnixAddressRegistered(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	var localBypassRegistryServer registry.NetworkServiceEndpointRegistryServer
	localBypassNetworkServiceServer := localbypass.NewServer(&localBypassRegistryServer)
	server := next.NewNetworkServiceServer(localBypassNetworkServiceServer, &testNetworkServiceServer{})

	srv := chain.NewNetworkServiceEndpointRegistryServer(localBypassRegistryServer, seturl.NewNetworkServiceEndpointRegistryServer("tcp:127.0.0.1:5002"))
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkServiceEndpointName: "nse-1",
		},
	}
	_, err := srv.Register(
		context.Background(),
		&registry.NetworkServiceEndpoint{
			Name: "nse-1",
			Url:  "unix:///var/run/nse-1.sock",
		})

	assert.Nil(t, err)

	ctx := withTestData(context.Background(), &TestData{})
	_, err = server.Request(ctx, request)
	assert.Nil(t, err)
	assert.Equal(t, &url.URL{Scheme: "unix", Path: "/var/run/nse-1.sock"}, testData(ctx).clientURL)

	ctx = withTestData(context.Background(), &TestData{})
	_, err = server.Close(ctx, request.GetConnection())
	assert.Nil(t, err)
	assert.Equal(t, &url.URL{Scheme: "unix", Path: "/var/run/nse-1.sock"}, testData(ctx).clientURL)
}

func TestNewServer_NonUnixAddressRegistered(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	var localBypassRegistryServer registry.NetworkServiceEndpointRegistryServer
	localBypassNetworkServiceServer := localbypass.NewServer(&localBypassRegistryServer)
	server := next.NewNetworkServiceServer(localBypassNetworkServiceServer, &testNetworkServiceServer{})
	srv := chain.NewNetworkServiceEndpointRegistryServer(localBypassRegistryServer, seturl.NewNetworkServiceEndpointRegistryServer("tcp:127.0.0.1:5002"))
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkServiceEndpointName: "nse-1",
		},
	}
	_, err := srv.Register(
		context.Background(),
		&registry.NetworkServiceEndpoint{
			Name: "nse-1",
			Url:  "tcp://127.0.0.1:5002",
		},
	)
	assert.Nil(t, err)

	ctx := withTestData(context.Background(), &TestData{})
	_, err = server.Request(ctx, request)
	assert.Nil(t, err)
	assert.Nil(t, err)
	assert.Equal(t, &url.URL{Scheme: "tcp", Host: "127.0.0.1:5002"}, testData(ctx).clientURL)

	ctx = withTestData(context.Background(), &TestData{})
	_, err = server.Close(ctx, request.GetConnection())
	assert.Nil(t, err)
	assert.Nil(t, err)
	assert.Equal(t, &url.URL{Scheme: "tcp", Host: "127.0.0.1:5002"}, testData(ctx).clientURL)
}

func TestNewServer_AddsNothingAfterNSERemoval(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	var localBypassRegistryServer registry.NetworkServiceEndpointRegistryServer
	localBypassNetworkServiceServer := localbypass.NewServer(&localBypassRegistryServer)
	server := next.NewNetworkServiceServer(localBypassNetworkServiceServer, &testNetworkServiceServer{})
	srv := chain.NewNetworkServiceEndpointRegistryServer(localBypassRegistryServer, seturl.NewNetworkServiceEndpointRegistryServer("tcp:127.0.0.1:5002"))
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkServiceEndpointName: "nse-1",
		},
	}
	_, err := srv.Register(
		peer.NewContext(context.Background(),
			&peer.Peer{
				Addr: &net.UnixAddr{
					Name: "/var/run/nse-1.sock",
					Net:  "unix",
				},
				AuthInfo: nil,
			}),
		&registry.NetworkServiceEndpoint{
			Name: "nse-1",
		},
	)
	assert.Nil(t, err)
	_, err = localBypassRegistryServer.Unregister(
		context.Background(),
		&registry.NetworkServiceEndpoint{
			Name: "nse-1",
		})
	assert.Nil(t, err)

	ctx := withTestData(context.Background(), &TestData{})
	_, err = server.Request(ctx, request)
	assert.Nil(t, err)
	assert.Nil(t, testData(ctx).clientURL)

	ctx = withTestData(context.Background(), &TestData{})
	_, err = server.Close(ctx, request.GetConnection())
	assert.Nil(t, err)
	assert.Nil(t, testData(ctx).clientURL)
}
