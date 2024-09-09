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

package injecterror

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type injectErrorNSEClient struct {
	registerErrorSupplier, findErrorSupplier, unregisterErrorSupplier *errorSupplier
}

// NewNetworkServiceEndpointRegistryClient returns a client chain element returning error on Register/Find/Unregister on given times.
func NewNetworkServiceEndpointRegistryClient(opts ...Option) registry.NetworkServiceEndpointRegistryClient {
	o := &options{
		err:                  errors.New("error originates in injectErrorNSEClient"),
		registerErrorTimes:   []int{-1},
		findErrorTimes:       []int{-1},
		unregisterErrorTimes: []int{-1},
	}

	for _, opt := range opts {
		opt(o)
	}

	return &injectErrorNSEClient{
		registerErrorSupplier: &errorSupplier{
			err:        o.err,
			errorTimes: o.registerErrorTimes,
		},
		findErrorSupplier: &errorSupplier{
			err:        o.err,
			errorTimes: o.findErrorTimes,
		},
		unregisterErrorSupplier: &errorSupplier{
			err:        o.err,
			errorTimes: o.unregisterErrorTimes,
		},
	}
}

func (c *injectErrorNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	if err := c.registerErrorSupplier.supply(); err != nil {
		return nil, err
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
}

func (c *injectErrorNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	if err := c.findErrorSupplier.supply(); err != nil {
		return nil, err
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *injectErrorNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	if err := c.unregisterErrorSupplier.supply(); err != nil {
		return nil, err
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}
