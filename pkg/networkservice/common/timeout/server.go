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

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/after"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type timeoutServer struct {
	ctx        context.Context
	afterFuncs afterFuncMap
}

// NewServer - creates a new NetworkServiceServer chain element that implements timeout of expired connections
//             for the subsequent chain elements.
// WARNING: `timeout` uses ctx as a context for the Close, so if there are any chain elements setting some data
//          in context in chain before the `timeout`, these changes won't appear in the Close context.
func NewServer(ctx context.Context) networkservice.NetworkServiceServer {
	return &timeoutServer{
		ctx: ctx,
	}
}

func (t *timeoutServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if err := t.validateRequest(ctx, request); err != nil {
		return nil, err
	}

	afterFunc, loaded := t.afterFuncs.Load(request.GetConnection().GetId())
	stopped := loaded && afterFunc.Stop()

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		if stopped {
			afterFunc.Resume()
		}
		return nil, err
	}

	if loaded {
		afterFunc.Reset(conn.GetPrevPathSegment().GetExpires().AsTime().Local())
	} else {
		t.startAfterFunc(ctx, conn.Clone())
	}

	return conn, nil
}

func (t *timeoutServer) validateRequest(ctx context.Context, request *networkservice.NetworkServiceRequest) error {
	if request.GetConnection().GetPrevPathSegment().GetExpires() == nil {
		return errors.Errorf("expiration for prev path segment cannot be nil. conn: %+v", request.GetConnection())
	}
	if serialize.GetExecutor(ctx) == nil {
		return errors.New("no executor provided")
	}
	return nil
}

func (t *timeoutServer) startAfterFunc(ctx context.Context, conn *networkservice.Connection) {
	logger := log.FromContext(ctx).WithField("timeoutServer", "startAfterFunc")

	t.afterFuncs.Store(conn.GetId(), after.NewFunc(t.ctx, conn.GetPrevPathSegment().GetExpires().AsTime().Local(), func() {
		<-serialize.GetExecutor(ctx).AsyncExec(func() {
			t.afterFuncs.Delete(conn.GetId())

			closeCtx, cancel := context.WithCancel(t.ctx)
			defer cancel()

			if _, err := next.Server(ctx).Close(closeCtx, conn); err != nil {
				logger.Errorf("failed to close timed out connection: %s %s", conn.GetId(), err.Error())
			}
		})
	}))
}

func (t *timeoutServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("timeoutServer", "Close")

	afterFunc, ok := t.afterFuncs.LoadAndDelete(conn.GetId())
	if !ok {
		logger.Warnf("connection has been already closed: %s", conn.GetId())
		return new(empty.Empty), nil
	}
	afterFunc.Stop()

	return next.Server(ctx).Close(ctx, conn)
}
