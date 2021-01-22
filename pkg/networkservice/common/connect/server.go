// Copyright (c) 2020 Cisco and/or its affiliates.
//
// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

// Package connect is intended to allow passthrough style Endpoints to have a server that also connects to a client
package connect

import (
	"context"
	"net/url"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/tools/clientmap"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
)

type connectServer struct {
	ctx               context.Context
	clientFactory     func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient
	clientDialOptions []grpc.DialOption
	connInfos         connectionInfoMap
	clients           clientmap.RefcountMap
}

type connectionInfo struct {
	clientURL *url.URL
	client    networkservice.NetworkServiceClient
}

// NewServer - chain element that
func NewServer(
	ctx context.Context,
	clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient,
	clientDialOptions ...grpc.DialOption,
) networkservice.NetworkServiceServer {
	return &connectServer{
		ctx:               ctx,
		clientFactory:     clientFactory,
		clientDialOptions: clientDialOptions,
	}
}

func (s *connectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := s.client(ctx, request.GetConnection()).Request(ctx, request.Clone())
	if err != nil {
		return nil, err
	}

	// Update request.Connection
	request.Connection = conn

	// Carry on with next.Server
	return next.Server(ctx).Request(ctx, request)
}

func (s *connectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	var clientErr error
	if connInfo, ok := s.connInfos.LoadAndDelete(conn.GetId()); ok {
		_, clientErr = connInfo.client.Close(ctx, conn)
	}

	_, err := next.Server(ctx).Close(ctx, conn)

	if clientErr != nil && err != nil {
		return nil, errors.Wrapf(err, "errors during client close: %v", clientErr)
	}
	if clientErr != nil {
		return nil, errors.Wrap(clientErr, "errors during client close")
	}
	return &empty.Empty{}, err
}

func (s *connectServer) client(ctx context.Context, conn *networkservice.Connection) networkservice.NetworkServiceClient {
	logEntry := logger.Log(ctx).WithField("connectServer", "client")

	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		err := errors.Errorf("clientURL not found for incoming connection: %+v", conn)
		return injecterror.NewClient(err)
	}

	// First check if we have already requested some clientURL with this conn.GetID().
	if connInfo, ok := s.connInfos.Load(conn.GetId()); ok {
		if *connInfo.clientURL == *clientURL {
			return connInfo.client
		}
		// For some reason we have changed the clientURL, so we need to close the existing client.
		if _, clientErr := connInfo.client.Close(ctx, conn); clientErr != nil {
			logEntry.Warnf("failed to close client: %+v", clientErr)
		}
	}

	// Fast path if we already have client for the clientURL, use it.
	client, loaded := s.clients.Load(clientURL.String())
	if !loaded {
		// If not, create and LoadOrStore a new one.
		newClient, cancel := s.newClient(clientURL)
		client, loaded = s.clients.LoadOrStore(clientURL.String(), newClient)
		if loaded {
			// No one will use `newClient`, it should be canceled.
			cancel()
		}
	}

	s.connInfos.Store(conn.GetId(), connectionInfo{
		clientURL: clientURL,
		client:    client,
	})

	return client
}

func (s *connectServer) newClient(clientURL *url.URL) (networkservice.NetworkServiceClient, context.CancelFunc) {
	clientPtr := new(networkservice.NetworkServiceClient)

	ctx, cancel := context.WithCancel(s.ctx)
	onClose := func() {
		if deleted := s.clients.Delete(clientURL.String()); deleted {
			cancel()
		}
	}

	*clientPtr = chain.NewNetworkServiceClient(
		&connectClient{onClose: onClose},
		clienturl.NewClient(clienturlctx.WithClientURL(ctx, clientURL), s.clientFactory, s.clientDialOptions...),
	)

	return *clientPtr, cancel
}
