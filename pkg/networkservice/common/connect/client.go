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
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
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
		ctx, cancel := context.WithCancel(u.ctx)
		u.ctx = ctx

		clockTime := clock.FromContext(u.ctx)

		clientURL := clienturlctx.ClientURL(u.ctx)
		if clientURL == nil {
			u.dialErr = errors.New("cannot dial nil clienturl.ClientURL(ctx)")
			cancel()
			return
		}

		dialCtx, dialCancel := clockTime.WithTimeout(u.ctx, u.dialTimeout)
		defer dialCancel()

		dialOptions := append(append([]grpc.DialOption{}, u.dialOptions...), grpc.WithReturnConnectionError())

		var cc *grpc.ClientConn
		cc, u.dialErr = grpc.DialContext(dialCtx, grpcutils.URLToTarget(clientURL), dialOptions...)
		if u.dialErr != nil {
			cancel()
			return
		}

		u.client = u.clientFactory(u.ctx, cc)

		select {
		case <-dialCtx.Done():
			u.dialErr = dialCtx.Err()
		case u.dialErr = <-u.monitor(dialCtx, cancel, cc):
		}
	})

	return u.dialErr
}

func (u *connectClient) monitor(dialCtx context.Context, cancel context.CancelFunc, cc *grpc.ClientConn) <-chan error {
	errCh := make(chan error)
	go func() {
		defer func() {
			cancel()
			_ = cc.Close()
		}()

		stream, err := grpc_health_v1.NewHealthClient(cc).Watch(u.ctx, &grpc_health_v1.HealthCheckRequest{
			Service: networkservice.ServiceNames(null.NewServer())[0],
		})
		if err != nil {
			errCh <- err
			return
		}

		var closed bool
		for {
			var resp *grpc_health_v1.HealthCheckResponse
			resp, err = stream.Recv()
			if err != nil {
				if !closed {
					errCh <- err
				}
				return
			}

			switch resp.Status {
			case grpc_health_v1.HealthCheckResponse_SERVING:
				if !closed {
					closed = true
					close(errCh)
				}
			default:
				if dialCtx.Err() != nil {
					return
				}
			}
		}
	}()
	return errCh
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
