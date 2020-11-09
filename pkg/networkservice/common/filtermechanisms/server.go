// Copyright (c) 2019-2020 VMware, Inc.
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

// Package filtermechanisms filters out remote mechanisms if communicating by remote url
// filters out local mechanisms otherwise.
package filtermechanisms

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/registry/common/interpose"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/common/endpointurls"
)

type filterMechanismsServer struct {
	nses endpointurls.Map
}

// NewServer - filters out remote mechanisms if connection is received from a unix file socket, otherwise filters
// out local mechanisms
func NewServer(registryServer *registry.NetworkServiceEndpointRegistryServer) networkservice.NetworkServiceServer {
	result := &filterMechanismsServer{}
	*registryServer = endpointurls.NewNetworkServiceEndpointRegistryServer(&result.nses)
	return result
}

func (f *filterMechanismsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	u := clienturlctx.ClientURL(ctx)
	if name, ok := f.nses.Load(*u); ok {
		if !interpose.Is(name) {
			request.MechanismPreferences = filterMechanismsByCls(request.GetMechanismPreferences(), cls.LOCAL)
		}
	} else {
		request.MechanismPreferences = filterMechanismsByCls(request.GetMechanismPreferences(), cls.REMOTE)
	}
	return next.Server(ctx).Request(ctx, request)
}

func (f *filterMechanismsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

func filterMechanismsByCls(mechanisms []*networkservice.Mechanism, mechanismCls string) []*networkservice.Mechanism {
	var result []*networkservice.Mechanism
	for _, mechanism := range mechanisms {
		if mechanism.Cls == mechanismCls {
			result = append(result, mechanism)
		}
	}
	return result
}
