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
	"sync"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type discoverCandidatesServer struct {
	choices   sync.Map
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
		var nses []*registry.NetworkServiceEndpoint
		if interdomain.Is(request.GetConnection().NetworkService) {
			if v, ok := d.choices.Load(request.Connection.GetPath().PathSegments[request.Connection.GetPath().Index-2].Id); ok {
				nses = v.([]*registry.NetworkServiceEndpoint)
				d.choices.Delete(request.Connection.GetPath().PathSegments[request.Connection.GetPath().Index-2].Id)
			} else {
				return nil, errors.New("request has wrong nse")
			}
		} else {
			stream, err := d.nseClient.Find(ctx, &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: nseName}})
			if err != nil {
				return nil, err
			}
			nses = registry.ReadNetworkServiceEndpointList(stream)
		}
		var selected *registry.NetworkServiceEndpoint
		for _, nse := range nses {
			if interdomain.Target(nse.Name) == interdomain.Target(nseName) {
				selected = nse
				break
			}
		}
		if selected == nil {
			return nil, errors.New("request has wrong nse")
		}
		u, err := url.Parse(selected.Url)
		if err != nil {
			return nil, err
		}
		return next.Server(ctx).Request(clienturl.WithClientURL(ctx, u), request)
	}
	nseName = ""
	nsName := request.GetConnection().GetNetworkService()
	if interdomain.Is(nsName) {
		nseName = "@" + interdomain.Domain(nsName)
	}
	nseStream, err := d.nseClient.Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name:                nseName,
			NetworkServiceNames: []string{interdomain.Target(request.GetConnection().GetNetworkService())},
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
	if interdomain.Is(nsName) {
		d.choices.Store(request.Connection.GetPath().PathSegments[request.Connection.GetPath().Index].Id, nseList)
	}
	return next.Server(ctx).Request(ctx, request)
}

func (d *discoverCandidatesServer) Close(context.Context, *networkservice.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}
