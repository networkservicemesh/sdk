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
	nseClient registry.NetworkServiceEndpointRegistryClient
	nsClient  registry.NetworkServiceRegistryClient
}

// NewServer - creates a new NetworkServiceServer that can discover possible candidates for providing a requested
//             Network Service and add it to the context.Context where it can be retrieved by Candidates(ctx)
func NewServer(nsClient registry.NetworkServiceRegistryClient, nseClient registry.NetworkServiceEndpointRegistryClient) networkservice.NetworkServiceServer {
	return &discoverCandidatesServer{
		nseClient: nseClient,
		nsClient:  nsClient,
	}
}

func (d *discoverCandidatesServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// TODO - handle case where NetworkServiceEndpoint is already set
	// if request.GetConnection().GetNetworkServiceEndpointName() != "" {
	//    TODO what to do in this case?
	// }

	nseStream, err := d.nseClient.Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: request.GetConnection().GetNetworkService(),
		},
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	nses := readNSEsSteam(nseStream)

	nsStream, err := d.nsClient.Find(ctx, &registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: request.GetConnection().GetNetworkService(),
		},
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	nss := readNSsSteam(nsStream)

	ctx = WithCandidates(ctx, nses, nss[0])
	return next.Server(ctx).Request(ctx, request)
}

func readNSsSteam(stream registry.NetworkServiceRegistry_FindClient) []*registry.NetworkService {
	var result []*registry.NetworkService
	for msg, err := stream.Recv(); true; msg, err = stream.Recv() {
		if err != nil {
			break
		}
		result = append(result, msg)
	}
	return result
}

func readNSEsSteam(stream registry.NetworkServiceEndpointRegistry_FindClient) []*registry.NetworkServiceEndpoint {
	var result []*registry.NetworkServiceEndpoint
	for msg, err := stream.Recv(); true; msg, err = stream.Recv() {
		if err != nil {
			break
		}
		result = append(result, msg)
	}
	return result
}

func (d *discoverCandidatesServer) Close(context.Context, *networkservice.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}
