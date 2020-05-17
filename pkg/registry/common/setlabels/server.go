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

type setLabelsBulkRegisterNSEServer struct {
	registry.NetworkServiceRegistry_BulkRegisterNSEServer
}

func (s *setLabelsBulkRegisterNSEServer) Send(request *registry.NSERegistration) error {
	labels := request.GetNetworkServiceEndpoint().GetLabels()
	if labels == nil {
		labels = make(map[string]string)
		request.GetNetworkServiceEndpoint().Labels = labels
	}
	labels["networkservicename"] = request.GetNetworkService().GetName()
	return s.NetworkServiceRegistry_BulkRegisterNSEServer.Send(request)
}

type setLabelsServer struct{}

// NewServer creates new instance of NetworkServiceRegistryServer with setting networkservicename label
func NewServer() registry.NetworkServiceRegistryServer {
	return &setLabelsServer{}
}

func (m *setLabelsServer) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	labels := request.GetNetworkServiceEndpoint().GetLabels()
	if labels == nil {
		labels = make(map[string]string)
		request.GetNetworkServiceEndpoint().Labels = labels
	}
	labels["networkservicename"] = request.GetNetworkService().GetName()
	return next.NetworkServiceRegistryServer(ctx).RegisterNSE(ctx, request)
}

func (m *setLabelsServer) BulkRegisterNSE(s registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	return next.NetworkServiceRegistryServer(s.Context()).BulkRegisterNSE(&setLabelsBulkRegisterNSEServer{s})
}

func (m *setLabelsServer) RemoveNSE(ctx context.Context, req *registry.RemoveNSERequest) (*empty.Empty, error) {
	return next.NetworkServiceRegistryServer(ctx).RemoveNSE(ctx, req)
}

var _ registry.NetworkServiceRegistryServer = &setLabelsServer{}
