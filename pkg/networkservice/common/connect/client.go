// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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

package connect

import (
	"context"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

type connectClient struct {
	ctx           context.Context
	dialTimeout   time.Duration
	clientFactory client.Factory
	dialOptions   []grpc.DialOption
	initOnce      sync.Once
	dialErr       error
	client        networkservice.NetworkServiceClient
}

func (u *connectClient) init() error {
	u.initOnce.Do(func() {
		clockTime := clock.FromContext(u.ctx)

		clientURL := clienturlctx.ClientURL(u.ctx)
		if clientURL == nil {
			u.dialErr = errors.New("cannot dial nil clienturl.ClientURL(ctx)")
			return
		}

		ctx, cancel := clockTime.WithTimeout(u.ctx, u.dialTimeout)
		defer cancel()

		dialOptions := append(append([]grpc.DialOption{}, u.dialOptions...), grpc.WithReturnConnectionError())

		var cc *grpc.ClientConn
		cc, u.dialErr = grpc.DialContext(ctx, grpcutils.URLToTarget(clientURL), dialOptions...)
		if u.dialErr != nil {
			return
		}

		u.client = u.clientFactory(u.ctx, cc)

		go func() {
			defer func() {
				_ = cc.Close()
			}()
			for cc.WaitForStateChange(u.ctx, cc.GetState()) {
				switch cc.GetState() {
				case connectivity.Connecting, connectivity.Idle, connectivity.Ready:
					continue
				default:
					return
				}
			}
		}()
	})

	return u.dialErr
}

func (u *connectClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if err := u.init(); err != nil {
		return nil, err
	}
	return u.client.Request(ctx, request, opts...)
}

func (u *connectClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if err := u.init(); err != nil {
		return nil, err
	}
	return u.client.Close(ctx, conn, opts...)
}
