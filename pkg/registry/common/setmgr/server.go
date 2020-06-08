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

// Package setmgr implements a chain elemenet to set NetworkServiceManager to registered endpoints.
package setmgr

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type setMgrServer struct {
	manager *registry.NetworkServiceManager
}

// NewServer creates new instance of NetworkServiceRegistryServer which set the passed NSMgr
func NewServer(manager *registry.NetworkServiceManager) registry.NetworkServiceRegistryServer {
	return &setMgrServer{
		manager: manager,
	}
}

func (n *setMgrServer) RegisterNSE(ctx context.Context, r *registry.NSERegistration) (*registry.NSERegistration, error) {
	r.NetworkServiceManager = n.manager
	return next.NetworkServiceRegistryServer(ctx).RegisterNSE(ctx, r)
}

func (n *setMgrServer) BulkRegisterNSE(s registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	return next.NetworkServiceRegistryServer(s.Context()).BulkRegisterNSE(s)
}

func (n setMgrServer) RemoveNSE(ctx context.Context, r *registry.RemoveNSERequest) (*empty.Empty, error) {
	return next.NetworkServiceRegistryServer(ctx).RemoveNSE(ctx, r)
}

var _ registry.NetworkServiceRegistryServer = &setMgrServer{}
