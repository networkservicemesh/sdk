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

package serialize

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type serializeServer struct {
	serializer
}

// NewServer returns a new serialize server chain element
func NewServer() networkservice.NetworkServiceServer {
	return &serializeServer{
		serializer{
			executors: map[string]*executor{},
		},
	}
}

func (s *serializeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (conn *networkservice.Connection, err error) {
	connID := request.GetConnection().GetId()
	logrus.Infof("REQUEST: %v", connID)
	return s.requestConnection(ctx, connID, func(requestCtx context.Context) (*networkservice.Connection, error) {
		return next.Server(ctx).Request(requestCtx, request)
	})
}

func (s *serializeServer) Close(ctx context.Context, conn *networkservice.Connection) (_ *empty.Empty, err error) {
	logrus.Infof("CLOSE: %v", conn.GetId())
	return s.closeConnection(ctx, conn.GetId(), func(closeCtx context.Context) (*empty.Empty, error) {
		return next.Server(ctx).Close(closeCtx, conn)
	})
}
