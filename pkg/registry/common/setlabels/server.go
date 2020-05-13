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

package setlabels

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type setLabels struct {
}

// NewServer creates new instance of NetworkServiceRegistryServer with setting networkservicename label
func NewServer() registry.NetworkServiceRegistryServer {
	return &setLabels{}
}

func (m *setLabels) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	labels := request.GetNetworkServiceEndpoint().GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["networkservicename"] = request.GetNetworkService().GetName()
	return next.NetworkServiceRegistryServer(ctx).RegisterNSE(ctx, request)
}

func (m *setLabels) BulkRegisterNSE(s registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	return next.NetworkServiceRegistryServer(s.Context()).BulkRegisterNSE(s)
}

func (m *setLabels) RemoveNSE(ctx context.Context, req *registry.RemoveNSERequest) (*empty.Empty, error) {
	return next.NetworkServiceRegistryServer(ctx).RemoveNSE(ctx, req)
}

var _ registry.NetworkServiceRegistryServer = &setLabels{}
