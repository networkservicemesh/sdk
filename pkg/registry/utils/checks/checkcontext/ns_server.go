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

package checkcontext

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

// NewNSServer - returns NetworkServiceRegistryServer that checks the context passed in from the previous NSServer in the chain
//             t - *testing.T used for the check
//             check - function that checks the context.Context
func NewNSServer(t *testing.T, check func(*testing.T, context.Context)) registry.NetworkServiceRegistryServer {
	return &checkContextNSServer{
		T:     t,
		check: check,
	}
}

type checkContextNSServer struct {
	*testing.T
	check func(*testing.T, context.Context)
}

func (c *checkContextNSServer) Register(ctx context.Context, service *registry.NetworkService) (*registry.NetworkService, error) {
	c.check(c.T, ctx)
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, service)
}

func (c *checkContextNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	c.check(c.T, server.Context())
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (c *checkContextNSServer) Unregister(ctx context.Context, service *registry.NetworkService) (*empty.Empty, error) {
	c.check(c.T, ctx)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, service)
}
