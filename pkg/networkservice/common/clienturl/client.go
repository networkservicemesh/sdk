// Copyright (c) 2020 Cisco and/or its affiliates.
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

package clienturl

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

type clientURLClient struct {
	ctx           context.Context
	clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient
	dialOptions   []grpc.DialOption
	initOnce      sync.Once
	dialErr       error
	client        networkservice.NetworkServiceClient
}

// NewClient - creates a Client that will using clienturl.ClientUrl(ctx) to extract a url, dial it to a cc, use that cc with the clientFactory to produce a new
//             client to which it passes through any Request or Close calls
// 	ctx	- is full lifecycle context, any started clients will be terminated by this context done.
func NewClient(ctx context.Context, clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient, dialOptions ...grpc.DialOption) networkservice.NetworkServiceClient {
	rv := &clientURLClient{
		ctx:           ctx,
		clientFactory: clientFactory,
		dialOptions:   dialOptions,
	}
	return rv
}

func (u *clientURLClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if err := u.init(); err != nil {
		return nil, err
	}
	return u.client.Request(ctx, request)
}

func (u *clientURLClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if err := u.init(); err != nil {
		return nil, err
	}
	return u.client.Close(ctx, conn)
}

func (u *clientURLClient) init() error {
	u.initOnce.Do(func() {
		clientURL := ClientURL(u.ctx)
		if clientURL == nil {
			u.dialErr = errors.New("cannot dial nil clienturl.ClientURL(ctx)")
			return
		}
		var cc *grpc.ClientConn
		cc, u.dialErr = grpc.DialContext(u.ctx, grpcutils.URLToTarget(clientURL), u.dialOptions...)
		if u.dialErr != nil {
			return
		}
		u.client = u.clientFactory(u.ctx, cc)
		go func() {
			<-u.ctx.Done()
			_ = cc.Close()
		}()
	})
	return u.dialErr
}
