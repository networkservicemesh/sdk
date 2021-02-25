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

package clienturl

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

type nseRegistryURLClient struct {
	ctx           context.Context
	clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceEndpointRegistryClient
	dialOptions   []grpc.DialOption
	initOnce      sync.Once
	dialErr       error
	client        registry.NetworkServiceEndpointRegistryClient
}

func (u *nseRegistryURLClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	if err := u.init(); err != nil {
		return nil, err
	}
	resp, err := u.client.Register(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, resp, opts...)
}

func (u *nseRegistryURLClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	if err := u.init(); err != nil {
		return nil, err
	}
	resp, err := u.client.Find(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
	}
	return resp, err
}

func (u *nseRegistryURLClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	if err := u.init(); err != nil {
		return nil, err
	}
	_, err := u.client.Unregister(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

// NewNetworkServiceEndpointRegistryClient - creates a Client that will using clienturl.ClientUrl(ctx) to extract a url, dial it to a cc, use that cc with the clientFactory to produce a new
//             client to which it passes through any Request or Close calls
func NewNetworkServiceEndpointRegistryClient(ctx context.Context, clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceEndpointRegistryClient, dialOptions ...grpc.DialOption) registry.NetworkServiceEndpointRegistryClient {
	return &nseRegistryURLClient{
		clientFactory: clientFactory,
		dialOptions:   dialOptions,
		ctx:           ctx,
	}
}

func (u *nseRegistryURLClient) init() error {
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
