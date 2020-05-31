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

package memory

import (
	"context"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
)

type memoryServiceDiscoveryServer struct {
	storage *Storage
}

func (d *memoryServiceDiscoveryServer) FindNetworkService(ctx context.Context, req *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	log.Entry(ctx).Println("Find NS %+v", req)
	service, ok := d.storage.NetworkServices.Load(req.NetworkServiceName)
	if !ok {
		return nil, errors.Errorf("network service %v is not found", req.NetworkServiceName)
	}
	NSMs := map[string]*registry.NetworkServiceManager{}
	var NSEs []*registry.NetworkServiceEndpoint
	d.storage.NetworkServiceEndpoints.Range(func(_ string, v *registry.NetworkServiceEndpoint) bool {
		if v.NetworkServiceName == req.NetworkServiceName {
			NSEs = append(NSEs, v)
		}
		return true
	})
	for _, nse := range NSEs {
		nsm, ok := d.storage.NetworkServiceManagers.Load(nse.NetworkServiceManagerName)
		if ok {
			NSMs[nsm.Name] = nsm
		}
	}
	return &registry.FindNetworkServiceResponse{
		Payload: service.GetPayload(),
		NetworkService: &registry.NetworkService{
			Name:    service.GetName(),
			Payload: service.GetPayload(),
			Matches: service.Matches,
		},
		NetworkServiceManagers:  NSMs,
		NetworkServiceEndpoints: NSEs,
	}, nil
}

// NewNetworkServiceDiscoveryServer returns new instance of NetworkServiceDiscoveryServer based on resource client
func NewNetworkServiceDiscoveryServer(storage *Storage) registry.NetworkServiceDiscoveryServer {
	return &memoryServiceDiscoveryServer{
		storage: storage,
	}
}

var _ registry.NetworkServiceDiscoveryServer = &memoryServiceDiscoveryServer{}
