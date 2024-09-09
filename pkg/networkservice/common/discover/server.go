// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2023 Cisco Systems, Inc.
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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/matchutils"
)

type discoverCandidatesServer struct {
	nseClient registry.NetworkServiceEndpointRegistryClient
	nsClient  registry.NetworkServiceRegistryClient
}

// NewServer - creates a new NetworkServiceServer that can discover possible candidates for providing a requested
//
//	Network Service and add it to the context.Context where it can be retrieved by Candidates(ctx)
func NewServer(nsClient registry.NetworkServiceRegistryClient, nseClient registry.NetworkServiceEndpointRegistryClient) networkservice.NetworkServiceServer {
	return &discoverCandidatesServer{
		nseClient: nseClient,
		nsClient:  nsClient,
	}
}

func (d *discoverCandidatesServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if clienturlctx.ClientURL(ctx) != nil {
		return next.Server(ctx).Request(ctx, request)
	}

	nseName := request.GetConnection().GetNetworkServiceEndpointName()
	if nseName != "" {
		nse, err := d.discoverNetworkServiceEndpoint(ctx, nseName)
		if err != nil {
			return nil, err
		}
		u, err := url.Parse(nse.GetUrl())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse url %s", nse.GetUrl())
		}
		return next.Server(ctx).Request(clienturlctx.WithClientURL(ctx, u), request)
	}
	ns, err := d.discoverNetworkService(ctx, request.GetConnection().GetNetworkService(), request.GetConnection().GetPayload())
	if err != nil {
		return nil, err
	}
	nses, err := d.discoverNetworkServiceEndpoints(ctx, ns, request.GetConnection().GetLabels())
	if err != nil {
		return nil, err
	}

	request.GetConnection().Payload = ns.GetPayload()

	return next.Server(ctx).Request(WithCandidates(ctx, nses, ns), request.Clone())
}

func (d *discoverCandidatesServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// Unlike Request, Close method should always call next element in chain
	// to make sure we clear resources in the current app.

	logger := log.FromContext(ctx).WithField("discoverCandidatesServer", "Close")

	if clienturlctx.ClientURL(ctx) != nil {
		return next.Server(ctx).Close(ctx, conn)
	}

	nseName := conn.GetNetworkServiceEndpointName()
	if nseName == "" {
		logger.Error("network_service_endpoint_name is not set")
		return next.Server(ctx).Close(ctx, conn)
	}

	nse, err := d.discoverNetworkServiceEndpoint(ctx, nseName)
	if err != nil {
		logger.Errorf("endpoint is not found: %v: %v", nseName, err)
		return next.Server(ctx).Close(ctx, conn)
	}

	u, err := url.Parse(nse.GetUrl())
	if err != nil {
		logger.Errorf("failed to parse url: %s: %v", nse.GetUrl(), err)
		return next.Server(ctx).Close(ctx, conn)
	}

	return next.Server(ctx).Close(clienturlctx.WithClientURL(ctx, u), conn)
}

func (d *discoverCandidatesServer) discoverNetworkServiceEndpoint(ctx context.Context, nseName string) (*registry.NetworkServiceEndpoint, error) {
	query := &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: nseName,
		},
	}

	nseRespStream, err := d.nseClient.Find(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find %s", query.String())
	}
	nseList := registry.ReadNetworkServiceEndpointList(nseRespStream)

	for _, nse := range nseList {
		if nse.GetName() == nseName {
			return nse, nil
		}
	}

	return nil, errors.Errorf("network service endpoint %v not found", nseName)
}

func (d *discoverCandidatesServer) discoverNetworkServiceEndpoints(ctx context.Context, ns *registry.NetworkService, nsLabels map[string]string) ([]*registry.NetworkServiceEndpoint, error) {
	clockTime := clock.FromContext(ctx)

	query := &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceNames: []string{ns.GetName()},
		},
	}

	nseRespStream, err := d.nseClient.Find(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find %s", query.String())
	}
	nseList := registry.ReadNetworkServiceEndpointList(nseRespStream)

	result := matchutils.MatchEndpoint(nsLabels, ns, validateExpirationTime(clockTime, nseList)...)
	if len(result) != 0 {
		return result, nil
	}

	return nil, errors.New("network service endpoint candidates not found")
}

func (d *discoverCandidatesServer) discoverNetworkService(ctx context.Context, name, payload string) (*registry.NetworkService, error) {
	query := &registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name:    name,
			Payload: payload,
		},
	}

	nsRespStream, err := d.nsClient.Find(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find %s", query.String())
	}
	nsList := registry.ReadNetworkServiceList(nsRespStream)

	for _, ns := range nsList {
		if ns.GetName() == name {
			return ns, nil
		}
	}

	return nil, errors.Errorf("network service %v is not found", name)
}
