// Copyright (c) 2024 Cisco Systems, Inc.
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

// Package nsemonitor provides a NetworkServiceClient chain element that monitors NSEs deleted from registry
package nsemonitor

import (
	"context"

	"github.com/edwarnicke/genericsync"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type nseMonitorClient struct {
	chainCtx     context.Context
	registryConn registry.NetworkServiceEndpointRegistryClient
	genericsync.Map[string, context.CancelFunc]
}

// NewClient - creates a new NetworkServiceClient chain element that monitors NSE it has a connection to.
// If it gets "deleted: true" event from the registry for this NSE it calls Close to close the connection.
// Original goal is to close leaked connections to the dead NSEs in vl3 Network (because vl3 network doesn't have healing).
func NewClient(ctx context.Context, registryConn registry.NetworkServiceEndpointRegistryClient) networkservice.NetworkServiceClient {
	return &nseMonitorClient{
		chainCtx:     ctx,
		registryConn: registryConn,
	}
}

func (s *nseMonitorClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	nseName := conn.NetworkServiceEndpointName
	eventFactory := begin.FromContext(ctx)

	if oldCancel, loaded := s.LoadAndDelete(nseName); loaded {
		oldCancel()
	}

	watchCtx, cancel := context.WithCancel(s.chainCtx)
	query := &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: nseName},
		Watch:                  true,
	}
	stream, streamErr := s.registryConn.Find(watchCtx, query, opts...)
	if streamErr != nil {
		cancel()
		return nil, streamErr
	}

	s.Store(nseName, cancel)
	go func() {
		for {
			event, recvErr := stream.Recv()
			if recvErr != nil {
				return
			}

			if event.Deleted {
				eventFactory.Close(begin.CancelContext(s.chainCtx))
				return
			}
		}
	}()

	return conn, nil
}

func (s *nseMonitorClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if cancel, loaded := s.LoadAndDelete(conn.NetworkServiceEndpointName); loaded {
		cancel()
	}

	return next.Client(ctx).Close(ctx, conn, opts...)
}
