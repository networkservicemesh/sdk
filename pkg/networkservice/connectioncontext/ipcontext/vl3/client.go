// Copyright (c) 2022-2024 Cisco and/or its affiliates.
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

// Package vl3 provides chain elements that manage ipcontext of request for vL3 networks.
// Depends on `begin`, `metadata` chain elements.
package vl3

import (
	"context"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type vl3Client struct {
	pool         *IPAM
	chainContext context.Context
}

// NewClient - returns a new vL3 client instance that manages connection.context.ipcontext for vL3 scenario.
//
//	Produces refresh on prefix update.
//	Requires begin and metadata chain elements.
func NewClient(chainContext context.Context, pool *IPAM) networkservice.NetworkServiceClient {
	if chainContext == nil {
		panic("chainContext can not be nil")
	}
	if pool == nil {
		panic("vl3IPAM pool can not be nil")
	}
	r := &vl3Client{
		chainContext: chainContext,
		pool:         pool,
	}

	return r
}

func (n *vl3Client) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if !n.pool.isInitialized() {
		return nil, errors.New("prefix pool is initializing")
	}
	eventFactory := begin.FromContext(ctx)
	if eventFactory == nil {
		return nil, errors.New("begin is required. Please add begin.NewClient() into chain")
	}
	cancelCtx, cancel := context.WithCancel(n.chainContext)

	if oldCancel, loaded := loadAndDeleteCancel(ctx); loaded {
		oldCancel()
	}

	storeCancel(ctx, cancel)

	unsubscribe := n.pool.Subscribe(func() {
		eventFactory.Request(begin.CancelContext(cancelCtx))
	})

	go func() {
		defer unsubscribe()

		select {
		case <-n.chainContext.Done():
		case <-cancelCtx.Done():
		}
	}()

	if request.GetConnection() == nil {
		request.Connection = new(networkservice.Connection)
	}
	conn := request.GetConnection()
	if conn.GetContext() == nil {
		conn.Context = new(networkservice.ConnectionContext)
	}
	if conn.GetContext().GetIpContext() == nil {
		conn.GetContext().IpContext = new(networkservice.IPContext)
	}

	address, prefix := n.pool.selfAddress().String(), n.pool.selfPrefix().String()

	addAddr(&conn.GetContext().GetIpContext().SrcIpAddrs, address)
	addRoute(&conn.GetContext().GetIpContext().DstRoutes, address, n.pool.selfAddress().IP.String())
	addRoute(&conn.GetContext().GetIpContext().DstRoutes, prefix, n.pool.selfAddress().IP.String())

	return next.Client(ctx).Request(ctx, request, opts...)
}

func (n *vl3Client) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if oldCancel, loaded := loadAndDeleteCancel(ctx); loaded {
		oldCancel()
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
