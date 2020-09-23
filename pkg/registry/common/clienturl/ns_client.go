// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

type nsRegistryURLClient struct {
	ctx           context.Context
	clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient
	dialOptions   []grpc.DialOption
	initOnce      sync.Once
	dialErr       error
	client        registry.NetworkServiceRegistryClient
}

func (u *nsRegistryURLClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	if err := u.init(); err != nil {
		return nil, err
	}
	resp, err := u.client.Register(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, resp, opts...)
}

func (u *nsRegistryURLClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	if err := u.init(); err != nil {
		return nil, err
	}
	resp, err := u.client.Find(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return next.NetworkServiceRegistryClient(ctx).Find(ctx, in, opts...)
	}
	return resp, err
}

func (u *nsRegistryURLClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	if err := u.init(); err != nil {
		return nil, err
	}
	_, err := u.client.Unregister(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
}

// NewNetworkServiceRegistryClient - creates a Client that will using clienturl.ClientUrl(ctx) to extract a url, dial it to a cc, use that cc with the clientFactory to produce a new
//             client to which it passes through any Request or Close calls
// 	ctx - context be alive all lifetime for client exists, canceling of it means client should terminate
func NewNetworkServiceRegistryClient(ctx context.Context, clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient, dialOptions ...grpc.DialOption) registry.NetworkServiceRegistryClient {
	return &nsRegistryURLClient{
		clientFactory: clientFactory,
		dialOptions:   dialOptions,
		ctx:           ctx,
	}
}

func (u *nsRegistryURLClient) init() error {
	u.initOnce.Do(func() {
		clientURL := clienturlctx.ClientURL(u.ctx)
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
