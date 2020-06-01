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

	"github.com/networkservicemesh/sdk/pkg/registry/common/custom"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
)

type networkServiceRegistryServer struct {
	registry.NetworkServiceRegistryServer
}

// NewNetworkServiceRegistryServer creates new instance of NetworkServiceRegistryServer which set the unique name for the endpoint on registration
func NewNetworkServiceRegistryServer() registry.NetworkServiceRegistryServer {
	return &networkServiceRegistryServer{
		NetworkServiceRegistryServer: custom.NewServer(
			func(ctx context.Context, r *registry.NSERegistration) (*registry.NSERegistration, error) {
				if r.NetworkServiceEndpoint.Name == "" {
					if r.NetworkService.Name == "" {
						return nil, errors.New("network service has empty name")
					}
					r.NetworkServiceEndpoint.Name = fmt.Sprintf("%v-%v", r.NetworkService.Name, uuid.New().String())
				}
				return r, nil
			}, nil, nil),
	}
}

var _ registry.NetworkServiceRegistryServer = &networkServiceRegistryServer{}
