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

package tail

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
)

// tailNetworkServiceRegistryClient is a simple implementation of registry.NetworkServiceRegistryClientWrapper that is called at the end
// of a chain to ensure that we never call a method on a nil object
type tailNetworkServiceRegistryClient struct{}

func (n *tailNetworkServiceRegistryClient) RegisterNSE(_ context.Context, request *registry.NSERegistration, _ ...grpc.CallOption) (*registry.NSERegistration, error) {
	return request, nil
}

func (n *tailNetworkServiceRegistryClient) BulkRegisterNSE(_ context.Context, _ ...grpc.CallOption) (registry.NetworkServiceRegistry_BulkRegisterNSEClient, error) {
	// TODO implement it
	return nil, nil
}

func (n *tailNetworkServiceRegistryClient) RemoveNSE(_ context.Context, _ *registry.RemoveNSERequest, _ ...grpc.CallOption) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

// NewTailNetworkServiceRegistryClient - creates new Tail NetworkServiceRegistryClient
func NewTailNetworkServiceRegistryClient() registry.NetworkServiceRegistryClient {
	return &tailNetworkServiceRegistryClient{}
}

var _ registry.NetworkServiceRegistryClient = &tailNetworkServiceRegistryClient{}
