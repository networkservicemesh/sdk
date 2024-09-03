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

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type injectErrorNSEServer struct {
	registerErrorSupplier, findErrorSupplier, unregisterErrorSupplier *errorSupplier
}

// NewNetworkServiceEndpointRegistryServer returns a server chain element returning error on Register/Find/Unregister on given times.
func NewNetworkServiceEndpointRegistryServer(opts ...Option) registry.NetworkServiceEndpointRegistryServer {
	o := &options{
		err:                  errors.New("error originates in injectErrorNSEServer"),
		registerErrorTimes:   []int{-1},
		findErrorTimes:       []int{-1},
		unregisterErrorTimes: []int{-1},
	}

	for _, opt := range opts {
		opt(o)
	}

	return &injectErrorNSEServer{
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

func (s *injectErrorNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if err := s.registerErrorSupplier.supply(); err != nil {
		return nil, err
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *injectErrorNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	if err := s.findErrorSupplier.supply(); err != nil {
		return err
	}
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *injectErrorNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if err := s.unregisterErrorSupplier.supply(); err != nil {
		return nil, err
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
