// Copyright (c) 2021 Cisco and/or its affiliates.
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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type beginServer struct {
	Map
}

// NewServer - returns a new begin server chain element
func NewServer() networkservice.NetworkServiceServer {
	return &beginServer{}
}

func (b *beginServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (conn *networkservice.Connection, err error) {
	// If we already have a entry from some previous begin.NewServer or begin.NewClient, use that
	// otherwise, LoadOrStore it and add it to the ctx
	entry := FromContext(ctx)
	if entry == nil {
		entry, _ = b.LoadOrStore(request.GetConnection().GetId(), newEventOriginator(ctx, b, nil))
		ctx = withEntry(ctx, entry)
	}
	recurse := false
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(err, "timeout in beginClient.Request")
	case <-entry.executor.AsyncExec(func() {
		// ctx may have expired or been canceled while we were waiting to run
		select {
		case <-ctx.Done():
			err = errors.Wrap(err, "timeout in beginClient.Request AsyncExecutor")
			return
		default:
		}
		// If entry has changed between when we started waiting our turn for this request and now.
		// that means someone has closed this *before* we could request it.
		// which means this request came in *after* the close
		// which means we should continue withEntry the request, via recursion
		currentEntry, loaded := b.LoadOrStore(request.GetConnection().GetId(), newEventOriginator(ctx, b, nil))
		if !loaded || entry != currentEntry {
			recurse = true
			return
		}
		req := request.Clone()
		conn, err = next.Client(entry.nextCtx).Request(ctx, request)
		if err != nil {
			return
		}
		entry.update(req, conn)
	}):
		if recurse {
			return b.Request(ctx, request)
		}
	}

	return conn, err
}

func (b *beginServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	// If we already have a entry from some previous begin.NewServer or begin.NewClient, use that
	// otherwise, Load it and add it to the ctx
	entry := FromContext(ctx)
	if entry == nil {
		var loaded bool
		entry, loaded = b.Load(conn.GetId())
		if !loaded || entry == nil {
			return &empty.Empty{}, nil
		}
		ctx = withEntry(ctx, entry)
	}
	var err error
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(err, "timeout in beginClient.Request")
	case <-entry.executor.AsyncExec(func() {
		// ctx may have expired or been canceled while we were waiting to run
		select {
		case <-ctx.Done():
			err = errors.Wrap(err, "timeout in beginClient.Request AsyncExecutor")
			return
		default:
		}
		// If entry has disappeared since we started waiting our turn it has been Closed already
		// If entry has changed since we started waiting our turn, then it has been closed and requested, and
		// is thus *not* the same request we were asked to Close
		// in either case, we should not continue to close
		currentEntry, loaded := b.Load(conn.GetId())
		if !loaded || entry != currentEntry {
			return
		}
		if entry.conn != nil {
			_, err = next.Client(ctx).Close(entry.nextCtx, entry.conn)
		}
	}):
	}

	return &empty.Empty{}, err
}
