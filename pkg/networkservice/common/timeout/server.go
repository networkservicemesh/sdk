// Copyright (c) 2020 Cisco Systems, Inc.
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

// Package timeout provides a NetworkServiceServer chain element that times out expired connection
package timeout

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/expire"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/serializectx"
)

type timeoutServer struct {
	expireManager *expire.Manager
}

// NewServer - creates a new NetworkServiceServer chain element that implements timeout of expired connections
//             for the subsequent chain elements.
func NewServer(ctx context.Context) networkservice.NetworkServiceServer {
	return &timeoutServer{
		expireManager: expire.NewManager(ctx),
	}
}

func (s *timeoutServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (conn *networkservice.Connection, err error) {
	logger := log.FromContext(ctx).WithField("timeoutServer", "Request")

	conn = request.GetConnection()
	connID := conn.GetId()

	expirationTimestamp := conn.GetPrevPathSegment().GetExpires()
	if expirationTimestamp == nil {
		return nil, errors.Errorf("expiration for the previous path segment cannot be nil: %+v", conn)
	}

	s.expireManager.Stop(connID)

	conn, err = next.Server(ctx).Request(ctx, request)
	if err != nil {
		s.expireManager.Start(connID)
		return nil, err
	}

	closeConn := conn.Clone()
	if closeConn.GetId() != connID {
		s.expireManager.Delete(connID)
	}

	s.expireManager.New(
		serializectx.GetExecutor(ctx, closeConn.GetId()),
		closeConn.GetId(),
		expirationTimestamp.AsTime().Local(),
		func(closeCtx context.Context) {
			if _, closeErr := next.Server(ctx).Close(closeCtx, closeConn); closeErr != nil {
				logger.Errorf("failed to close timed out connection: %s %s", closeConn.GetId(), closeErr.Error())
			}
		},
	)

	return conn, nil
}

func (s *timeoutServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("timeoutServer", "Close")

	if s.expireManager.Stop(conn.GetId()) {
		if _, err := next.Server(ctx).Close(ctx, conn); err != nil {
			s.expireManager.Start(conn.GetId())
			return nil, err
		}
		s.expireManager.Delete(conn.GetId())
	} else {
		logger.Warnf("connection has been already closed: %s", conn.GetId())
	}

	return new(empty.Empty), nil
}
