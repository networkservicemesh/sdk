// Copyright (c) 2023 Cisco and/or its affiliates.
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

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type injectErrorNSServer struct {
	registerErrorSupplier, findErrorSupplier, unregisterErrorSupplier *errorSupplier
}

// NewNetworkServiceRegistryServer returns a server chain element returning error on Register/Find/Unregister on given times.
func NewNetworkServiceRegistryServer(opts ...Option) registry.NetworkServiceRegistryServer {
	o := &options{
		err:                  errors.New("error originates in injectErrorNSServer"),
		registerErrorTimes:   []int{-1},
		findErrorTimes:       []int{-1},
		unregisterErrorTimes: []int{-1},
	}

	for _, opt := range opts {
		opt(o)
	}

	return &injectErrorNSServer{
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

func (c *injectErrorNSServer) Register(ctx context.Context, in *registry.NetworkService) (*registry.NetworkService, error) {
	if err := c.registerErrorSupplier.supply(); err != nil {
		return nil, err
	}
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, in)
}

func (c *injectErrorNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	if err := c.findErrorSupplier.supply(); err != nil {
		return err
	}
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (c *injectErrorNSServer) Unregister(ctx context.Context, in *registry.NetworkService) (*empty.Empty, error) {
	if err := c.unregisterErrorSupplier.supply(); err != nil {
		return nil, err
	}
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, in)
}
