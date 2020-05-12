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
	"github.com/pkg/errors"
)

type tailRegistryNSMServer struct{}

func (t tailRegistryNSMServer) RegisterNSM(_ context.Context, req *registry.NetworkServiceManager) (*registry.NetworkServiceManager, error) {
	return req, nil
}

func (t tailRegistryNSMServer) GetEndpoints(context.Context, *empty.Empty) (*registry.NetworkServiceEndpointList, error) {
	return nil, errors.New("network service endpoints are not found")
}

var _ registry.NsmRegistryServer = &tailRegistryNSMServer{}
