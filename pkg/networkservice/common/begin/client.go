// Copyright (c) 2021-2024 Cisco and/or its affiliates.
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

package begin

import (
	"context"

	"github.com/edwarnicke/genericsync"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type beginClient struct {
	genericsync.Map[string, *eventFactoryClient]
}

// NewClient - creates a new begin chain element
func NewClient() networkservice.NetworkServiceClient {
	return &beginClient{}
}

func (b *beginClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (conn *networkservice.Connection, err error) {
	// No connection.ID, no service
	if request.GetConnection().GetId() == "" {
		return nil, errors.New("request.EventFactory.Id must not be zero valued")
	}
	// If some other EventFactory is already in the ctx... we are already running in an executor, and can just execute normally
	if fromContext(ctx) != nil {
		return next.Client(ctx).Request(ctx, request, opts...)
	}
	eventFactoryClient, _ := b.LoadOrStore(request.GetConnection().GetId(),
		newEventFactoryClient(
			ctx,
			func() *eventFactoryClient {
				currentEventFactoryClient, _ := b.Load(request.GetConnection().GetId())
				return currentEventFactoryClient
			},
			func() {
				b.Delete(request.GetRequestConnection().GetId())
			},
		),
	)
	err = <-eventFactoryClient.Request(
		withContext(ctx),
		withUserRequest(request),
		withGRPCOpts(opts),
		withConnectionToReturn(&conn),
	)
	if err != nil {
		if errors.Is(err, &errorEventFactoryInconsistency{}) {
			log.FromContext(ctx).Debug("recalling begin.Request because currentEventFactoryClient != eventFactoryClient")
			conn, err = b.Request(ctx, request, opts...)
		}
	}

	return conn, err
}

func (b *beginClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (emp *emptypb.Empty, err error) {
	// If some other EventFactory is already in the ctx... we are already running in an executor, and can just execute normally
	if fromContext(ctx) != nil {
		return next.Client(ctx).Close(ctx, conn, opts...)
	}
	eventFactoryClient, ok := b.Load(conn.GetId())
	if !ok {
		// If we don't have a connection to Close, just let it be
		return
	}
	err = <-eventFactoryClient.Close(
		withGRPCOpts(opts),
	)

	return emp, err
}
