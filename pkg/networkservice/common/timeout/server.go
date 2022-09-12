// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2022 Cisco Systems, Inc.
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
	"time"

	iserror "errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type timeoutServer struct {
	chainCtx context.Context
}

// NewServer - creates a new NetworkServiceServer chain element that implements timeout of expired connections
//
//	for the subsequent chain elements.
func NewServer(ctx context.Context) networkservice.NetworkServiceServer {
	return &timeoutServer{
		chainCtx: ctx,
	}
}

func (s *timeoutServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (conn *networkservice.Connection, err error) {
	conn, err = next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	expirationTimestamp := conn.GetPrevPathSegment().GetExpires()
	if expirationTimestamp == nil {
		return nil, errors.Errorf("expiration for the previous path segment cannot be nil: %+v", conn)
	}
	expirationTime := expirationTimestamp.AsTime()
	cancelCtx, cancel := context.WithCancel(s.chainCtx)
	if oldCancel, loaded := loadAndDelete(ctx, metadata.IsClient(s)); loaded {
		oldCancel()
	}
	store(ctx, metadata.IsClient(s), cancel)
	eventFactory := begin.FromContext(ctx)
	timeClock := clock.FromContext(ctx)
	afterCh := timeClock.After(timeClock.Until(expirationTime))
	go func(cancelCtx context.Context, afterCh <-chan time.Time) {
		select {
		case <-cancelCtx.Done():
		case <-afterCh:
			eventFactory.Close(begin.CancelContext(cancelCtx))
		}
	}(cancelCtx, afterCh)

	return conn, nil
}

func (s *timeoutServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, err := next.Server(ctx).Close(ctx, conn)
	if !(iserror.Is(err, context.DeadlineExceeded) || iserror.Is(err, context.Canceled)) {
		if oldCancel, loaded := loadAndDelete(ctx, metadata.IsClient(s)); loaded {
			oldCancel()
		}
	}
	return &empty.Empty{}, err
}
