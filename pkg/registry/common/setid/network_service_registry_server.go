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

package setid

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type networkServiceRegistryServer struct{}

// NewNetworkServiceRegistryServer creates new instance of NetworkServiceRegistryServer which set the unique name for the endpoint on registration
func NewNetworkServiceRegistryServer() registry.NetworkServiceRegistryServer {
	return &networkServiceRegistryServer{}
}

func (n networkServiceRegistryServer) RegisterNSE(ctx context.Context, r *registry.NSERegistration) (*registry.NSERegistration, error) {
	if r.NetworkServiceEndpoint.Name == "" {
		if r.NetworkService.Name == "" {
			return nil, errors.New("network service has empty name")
		}
		r.NetworkServiceEndpoint.Name = fmt.Sprintf("%v-%v", r.NetworkService.Name, uuid.New().String())
	}
	return next.NetworkServiceRegistryServer(ctx).RegisterNSE(ctx, r)
}

func (n *networkServiceRegistryServer) BulkRegisterNSE(s registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	return next.NetworkServiceRegistryServer(s.Context()).BulkRegisterNSE(s)
}

func (n networkServiceRegistryServer) RemoveNSE(ctx context.Context, r *registry.RemoveNSERequest) (*empty.Empty, error) {
	return next.NetworkServiceRegistryServer(ctx).RemoveNSE(ctx, r)
}

var _ registry.NetworkServiceRegistryServer = &networkServiceRegistryServer{}
