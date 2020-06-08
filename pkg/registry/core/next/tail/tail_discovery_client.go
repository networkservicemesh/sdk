// Copyright (c) 2020 Cisco Systems, Inc.
//
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

// Package next provides a mechanism for chained registry.{Registry,Discovery}{Server,Client}s to call
// the next element in the chain.

package tail

import (
	"context"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type tailDiscoveryClient struct {
}

func (t tailDiscoveryClient) FindNetworkService(_ context.Context, request *registry.FindNetworkServiceRequest, _ ...grpc.CallOption) (*registry.FindNetworkServiceResponse, error) {
	return &registry.FindNetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name: request.NetworkServiceName,
		},
	}, nil
}

// NewTailDiscoveryClient - create tail NewTailDiscoveryClient
func NewTailDiscoveryClient() registry.NetworkServiceDiscoveryClient {
	return &tailDiscoveryClient{}
}

var _ registry.NetworkServiceDiscoveryClient = &tailDiscoveryClient{}
