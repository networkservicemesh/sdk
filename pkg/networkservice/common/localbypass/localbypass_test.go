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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/localbypass"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type testNetworkServiceServer struct {
	t    *testing.T
	want *url.URL
}

func (s testNetworkServiceServer) Request(ctx context.Context, in *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	assert.Equal(s.t, s.want, clienturl.ClientURL(ctx))
	return in.GetConnection(), nil
}

func (s testNetworkServiceServer) Close(ctx context.Context, _ *networkservice.Connection) (*empty.Empty, error) {
	assert.Equal(s.t, s.want, clienturl.ClientURL(ctx))
	return &empty.Empty{}, nil
}

func TestLocalBypassServer(t *testing.T) {
	var localBypassRegistryServer registry.NetworkServiceRegistryServer
	localBypassNetworkServiceServer := localbypass.NewServer(&localBypassRegistryServer)
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkServiceEndpointName: "nse-1",
		},
	}

	t.Run("adds nothing when there is no such NSE in sockets map", func(t *testing.T) {
		testRequest(t, localBypassNetworkServiceServer, request, nil)
	})

	_, _ = localBypassRegistryServer.RegisterNSE(
		peer.NewContext(context.Background(),
			&peer.Peer{
				Addr: &net.UnixAddr{
					Name: "/var/run/nse-1.sock",
					Net:  "unix",
				},
				AuthInfo: nil,
			}),
		&registry.NSERegistration{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name: "nse-1",
			},
		})
	t.Run("adds valid url after valid NSE registration with unix address type", func(t *testing.T) {
		testRequest(t, localBypassNetworkServiceServer, request, &url.URL{
			Scheme: "unix",
			Path:   "/var/run/nse-1.sock",
		})
	})

	_, _ = localBypassRegistryServer.RemoveNSE(
		context.Background(),
		&registry.RemoveNSERequest{
			NetworkServiceEndpointName: "nse-1",
		})
	t.Run("adds nothing after NSE removal", func(t *testing.T) {
		testRequest(t, localBypassNetworkServiceServer, request, nil)
	})

	_, _ = localBypassRegistryServer.RegisterNSE(
		peer.NewContext(context.Background(),
			&peer.Peer{
				Addr: &net.IPAddr{
					IP: net.IP{255, 255, 255, 255},
				},
				AuthInfo: nil,
			}),
		&registry.NSERegistration{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name: "nse-1",
			},
		})
	t.Run("adds nothing after invalid NSE registration with non-unix address", func(t *testing.T) {
		testRequest(t, localBypassNetworkServiceServer, request, nil)
	})
}

func testRequest(t *testing.T, s networkservice.NetworkServiceServer, request *networkservice.NetworkServiceRequest, want *url.URL) {
	server := next.NewNetworkServiceServer(s, &testNetworkServiceServer{t, want})
	_, _ = server.Request(context.Background(), request)
	_, _ = server.Close(context.Background(), request.GetConnection())
}
