// Copyright (c) 2022 Cisco and/or its affiliates.
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
	"errors"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/ipam"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type vl3Client struct {
	pool          vl3IPAM
	chainContext  context.Context
	executor      serialize.Executor
	subscriptions []chan struct{}
}

// NewClient - returns a new vL3 client instance that manages connection.context.ipcontext for vL3 scenario.
//
//	Produces refresh on prefix update.
//	Requires begin and metdata chain elements.
func NewClient(chainContext context.Context, prefixCh <-chan *ipam.PrefixResponse) networkservice.NetworkServiceClient {
	if chainContext == nil {
		panic("chainContext can not be nil")
	}
	if prefixCh == nil {
		panic("prefixCh can not be nil")
	}
	var r = &vl3Client{
		chainContext: chainContext,
	}

	go func() {
		for update := range prefixCh {
			prefixResp := update
			r.executor.AsyncExec(func() {
				r.pool.reset(chainContext, prefixResp.GetPrefix(), prefixResp.GetExcludePrefixes())
				for _, sub := range r.subscriptions {
					sub <- struct{}{}
				}
			})
		}
	}()

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

	notifyCh := make(chan struct{})

	n.executor.AsyncExec(func() {
		n.subscriptions = append(n.subscriptions, notifyCh)
	})

	go func() {
		defer func() {
			n.executor.AsyncExec(func() {
				for i, sub := range n.subscriptions {
					if sub == notifyCh {
						n.subscriptions = append(n.subscriptions[:i], n.subscriptions[i+1:]...)
						close(notifyCh)
						return
					}
				}
			})
		}()

		select {
		case <-n.chainContext.Done():
			return
		case <-cancelCtx.Done():
			return
		case <-notifyCh:
			eventFactory.Request(begin.CancelContext(cancelCtx))
		}
	}()

	if request.Connection == nil {
		request.Connection = new(networkservice.Connection)
	}
	var conn = request.GetConnection()
	if conn.GetContext() == nil {
		conn.Context = new(networkservice.ConnectionContext)
	}
	if conn.GetContext().GetIpContext() == nil {
		conn.GetContext().IpContext = new(networkservice.IPContext)
	}

	var address, prefix = n.pool.selfAddress().String(), n.pool.selfPrefix().String()

	conn.GetContext().GetIpContext().SrcIpAddrs = []string{address}
	conn.GetContext().GetIpContext().DstRoutes = []*networkservice.Route{
		{
			Prefix:  address,
			NextHop: n.pool.selfAddress().IP.String(),
		},
		{
			Prefix:  prefix,
			NextHop: n.pool.selfAddress().IP.String(),
		},
	}

	return next.Client(ctx).Request(ctx, request, opts...)
}

func (n *vl3Client) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if oldCancel, loaded := loadAndDeleteCancel(ctx); loaded {
		oldCancel()
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
