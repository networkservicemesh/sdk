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

package vl3mtu

import (
	"context"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/upstreamrefresh"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type vl3MtuClient struct {
	minMtu uint32
	m      sync.Mutex
}

// NewClient - returns a new vl3mtu client chain element.
// It stores a minimum mtu of the vl3 mtu and updates connection context if required.
func NewClient() networkservice.NetworkServiceClient {
	return &vl3MtuClient{
		minMtu: jumboFrameSize,
	}
}

func (v *vl3MtuClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	c := request.GetConnection()
	if c.GetContext() == nil {
		c.Context = &networkservice.ConnectionContext{}
	}

	// Check MTU of the connection
	minMTU := atomic.LoadUint32(&v.minMtu)
	if minMTU < c.GetContext().GetMTU() || c.GetContext().GetMTU() == 0 {
		c.GetContext().MTU = minMTU
	}

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	// Update MTU of the vl3
	if v.updateMinMTU(conn) {
		if ln, ok := upstreamrefresh.LoadLocalNotifier(ctx, metadata.IsClient(v)); ok {
			ln.Notify(ctx, conn.GetId())
		}
	}

	return conn, nil
}

func (v *vl3MtuClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

// updateMinMTU - returns true if mtu was updated.
func (v *vl3MtuClient) updateMinMTU(conn *networkservice.Connection) bool {
	if atomic.LoadUint32(&v.minMtu) <= conn.GetContext().GetMTU() {
		return false
	}

	v.m.Lock()
	defer v.m.Unlock()
	if atomic.LoadUint32(&v.minMtu) <= conn.GetContext().GetMTU() {
		return false
	}
	atomic.StoreUint32(&v.minMtu, conn.GetContext().GetMTU())
	return true
}
