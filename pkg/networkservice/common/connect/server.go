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
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

// ClientFactory is used to created new clients when new connection is created.
type ClientFactory = func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient

type connectServer struct {
	ctx               context.Context
	clientFactory     ClientFactory
	clientDialTimeout time.Duration
	clientDialOptions []grpc.DialOption
}

type connectionInfo struct {
	clientURL *url.URL
	cancel    context.CancelFunc
}

// NewServer - server chain element that creates client subchains and requests them selecting by
//             clienturlctx.ClientURL(ctx)
func NewServer(
	ctx context.Context,
	clientFactory ClientFactory,
	options ...Option,
) networkservice.NetworkServiceServer {
	s := &connectServer{
		ctx:               ctx,
		clientFactory:     clientFactory,
		clientDialTimeout: time.Second,
	}

	for _, opt := range options {
		opt(s)
	}

	return s
}

func (s *connectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (conn *networkservice.Connection, err error) {
	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		return nil, errors.Errorf("clientURL not found for incoming connection: %+v", request.GetConnection())
	}

	var connInfo *connectionInfo
	connInfo, err = s.connInfo(ctx, request.GetConnection())
	if err != nil {
		return nil, err
	}

	var client *connectClient
	client, connInfo.cancel = s.client(clientURL)
	defer func() {
		if err != nil {
			connInfo.cancel()
		}
	}()

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err = client.Request(ctx, request.Clone())
	if err != nil {
		return nil, err
	}

	request.Connection = conn

	if conn, err = next.Server(ctx).Request(ctx, request); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		_, closeErr := client.Close(closeCtx, request.Connection.Clone())
		if closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		connInfo.cancel()
	}

	return conn, err
}

func (s *connectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	var clientErr error
	if connInfo, ok := load(ctx); ok {
		connInfo.cancel()

		client, cancel := s.client(connInfo.clientURL)
		defer cancel()

		_, clientErr = client.Close(ctx, conn)
	} else {
		clientErr = errors.Errorf("no client found for the connection: %s", conn.GetId())
	}

	_, err := next.Server(ctx).Close(ctx, conn)

	if clientErr != nil && err != nil {
		return nil, errors.Wrapf(err, "errors during client close: %s", clientErr.Error())
	}
	if clientErr != nil {
		return nil, errors.Wrap(clientErr, "errors during client close")
	}
	return &empty.Empty{}, err
}

func (s *connectServer) connInfo(ctx context.Context, conn *networkservice.Connection) (*connectionInfo, error) {
	logger := log.FromContext(ctx).WithField("connectServer", "connInfo")

	clientURL := clienturlctx.ClientURL(ctx)

	connInfo, loaded := loadOrStore(ctx, &connectionInfo{
		clientURL: clientURL,
		cancel:    func() {},
	})
	if !loaded {
		return connInfo, nil
	}

	connInfo.cancel()

	if connInfo.clientURL.String() != clientURL.String() {
		client, cancel := s.client(connInfo.clientURL)
		defer cancel()

		if _, clientErr := client.Close(ctx, conn.Clone()); clientErr != nil {
			logger.Warnf("failed to close client: %s", clientErr.Error())
		}

		if err := ctx.Err(); err != nil {
			return nil, err
		}

		connInfo.clientURL = clientURL
	}

	return connInfo, nil
}

func (s *connectServer) client(clientURL *url.URL) (*connectClient, context.CancelFunc) {
	ctx, cancel := context.WithCancel(s.ctx)
	return &connectClient{
		ctx:           clienturlctx.WithClientURL(ctx, clientURL),
		dialTimeout:   s.clientDialTimeout,
		clientFactory: s.clientFactory,
		dialOptions:   s.clientDialOptions,
	}, cancel
}
