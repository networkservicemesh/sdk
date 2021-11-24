// Copyright (c) 2020 Cisco and/or its affiliates.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package metadata

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type metadataServer struct {
	Map metaDataMap
}

// NewServer - Enable per Connection.Id metadata for the server
//             Must come after updatepath.NewServer() in the chain
func NewServer() networkservice.NetworkServiceServer {
	return &metadataServer{}
}

func (m *metadataServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	connID := request.GetConnection().GetId()
	_, isEstablished := m.Map.Load(connID)

	conn, err := next.Server(ctx).Request(store(ctx, connID, &m.Map), request)
	if err != nil {
		if !isEstablished {
			del(ctx, connID, &m.Map)

			log.FromContext(ctx).
				WithField("metadata", "server").
				WithField("connID", connID).
				Debugf("metadata deleted")
		}
		return nil, err
	}

	return conn, nil
}

func (m *metadataServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	delCtx := del(ctx, conn.GetId(), &m.Map)

	log.FromContext(ctx).
		WithField("metadata", "server").
		WithField("connID", conn.GetId()).
		Debugf("metadata deleted")

	return next.Server(ctx).Close(delCtx, conn)
}
