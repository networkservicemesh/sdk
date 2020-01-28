// Copyright (c) 2020 Cisco Systems, Inc.
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

package discover

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type discoverCandidatesServer struct {
	registry registry.NetworkServiceDiscoveryClient
}

// NewServer - creates a new NetworkServiceServer that can discover possible candidates for providing a requested
//             Network Service and add it to the context.Context where it can be retrieved by Candidates(ctx)
func NewServer(reg registry.NetworkServiceDiscoveryClient) networkservice.NetworkServiceServer {
	return &discoverCandidatesServer{registry: reg}
}

func (d *discoverCandidatesServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// TODO - handle case where NetworkServiceEndpoint is already set
	// if request.GetConnection().GetNetworkServiceEndpointName() != "" {
	//    TODO what to do in this case?
	// }
	registryRequest := &registry.FindNetworkServiceRequest{
		NetworkServiceName: request.GetConnection().GetNetworkService(),
	}
	registryResponse, err := d.registry.FindNetworkService(ctx, registryRequest)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	registryResponse.NetworkServiceEndpoints = matchEndpoint(request.GetConnection().GetLabels(), registryResponse.GetNetworkService(), registryResponse.GetNetworkServiceEndpoints())
	// TODO handle local case
	ctx = WithCandidates(ctx, registryResponse)
	return next.Server(ctx).Request(ctx, request)
}

func (d *discoverCandidatesServer) Close(context.Context, *networkservice.Connection) (*empty.Empty, error) {
	panic("implement me")
}
