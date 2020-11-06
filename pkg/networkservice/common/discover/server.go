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
	"net/url"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

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
	nseName := request.GetConnection().GetNetworkServiceEndpointName()
	if nseName != "" {
		u, err := d.urlByNseName(nseName)
		if err != nil {
			return nil, err
		}
		return next.Server(ctx).Request(clienturlctx.WithClientURL(ctx, u), request)
	}
	nseStream, err := d.nseClient.Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceNames: []string{request.GetConnection().GetNetworkService()},
		},
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	nseList := registry.ReadNetworkServiceEndpointList(nseStream)

	nsStream, err := d.nsClient.Find(ctx, &registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: request.GetConnection().GetNetworkService(),
		},
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	nsList := registry.ReadNetworkServiceList(nsStream)
	if len(nsList) == 0 {
		return nil, errors.Errorf("network service %s is not found", request.GetConnection().GetNetworkService())
	}
	nseList = matchEndpoint(request.GetConnection().GetLabels(), nsList[0], nseList)

	if len(nseList) == 0 {
		return nil, errors.Errorf("network service endpoint for service %s is not found", request.GetConnection().GetNetworkService())
	}

	ctx = WithCandidates(ctx, nseList, nsList[0])
	return next.Server(ctx).Request(ctx, request)
}

func (d *discoverCandidatesServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	nseName := conn.GetNetworkServiceEndpointName()
	if nseName == "" {
		// If it's an existing connection, the NSE name should be set. Otherwise, it's probably an API misuse.
		return nil, errors.Errorf("network_service_endpoint_name is not set")
	}
	u, err := d.urlByNseName(nseName)
	if err != nil {
		return nil, err
	}
	return next.Server(ctx).Close(clienturlctx.WithClientURL(ctx, u), conn)
}

func (d *discoverCandidatesServer) urlByNseName(nseName string) (*url.URL, error) {
	nseStream, err := d.nseClient.Find(context.Background(), &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: nseName,
		},
	})
	if err != nil {
		return nil, err
	}
	nseList := registry.ReadNetworkServiceEndpointList(nseStream)
	if len(nseList) == 0 {
		return nil, errors.Errorf("network service endpoint %s is not found", nseName)
	}
	u, err := url.Parse(nseList[0].Url)
	if err != nil {
		return nil, err
	}
	return u, nil
}
