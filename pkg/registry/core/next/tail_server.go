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

package next

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

// tailNetworkServiceRegistryServer is a simple implementation of registry.NetworkServiceRegistryServer that is called at the end
// of a chain to ensure that we never call a method on a nil object
type tailNetworkServiceRegistryServer struct{}

func (t tailNetworkServiceRegistryServer) Register(_ context.Context, in *registry.NetworkServiceEntry) (*registry.NetworkServiceEntry, error) {
	return in, nil
}

func (t tailNetworkServiceRegistryServer) Find(*registry.NetworkServiceQuery, registry.NetworkServiceRegistry_FindServer) error {
	return nil
}

func (t tailNetworkServiceRegistryServer) Unregister(context.Context, *registry.NetworkServiceEntry) (*empty.Empty, error) {
	return new(empty.Empty), nil
}

var _ registry.NetworkServiceRegistryServer = &tailNetworkServiceRegistryServer{}
