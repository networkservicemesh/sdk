// Copyright (c) 2023-2024 Cisco Systems, Inc.
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

// Package netsvcmonitor provides a NetworkServiceServer chain element to provide a possible change nse for the connection immediately if network service was updated.
package netsvcmonitor

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/matchutils"
)

type monitorServer struct {
	chainCtx  context.Context
	nsClient  registry.NetworkServiceRegistryClient
	nseClient registry.NetworkServiceEndpointRegistryClient
}

// NewServer creates a new instance of netsvcmonitor server that allowes to the server chain monitor changes in the network service.
func NewServer(chainCtx context.Context, nsClient registry.NetworkServiceRegistryClient, nseClient registry.NetworkServiceEndpointRegistryClient) networkservice.NetworkServiceServer {
	return &monitorServer{
		chainCtx:  chainCtx,
		nsClient:  nsClient,
		nseClient: nseClient,
	}
}

func (m *monitorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if cancel, ok := loadCancelFunction(ctx); ok {
		cancel()
	}

	resp, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return resp, err
	}

	monitorCtx, cancel := context.WithCancel(m.chainCtx)
	minT := time.Time{}

	for _, seg := range resp.GetPath().GetPathSegments() {
		t := seg.Expires.AsTime().Local()
		if minT.After(t) || minT.IsZero() {
			minT = t
		}
	}

	if !minT.IsZero() {
		cancel()
		monitorCtx, cancel = context.WithTimeout(m.chainCtx, time.Until(minT))
	}

	storeCancelFunction(ctx, cancel)

	go m.monitorNetworkService(monitorCtx, resp.Clone(), begin.FromContext(ctx))

	return resp, err
}

func (m *monitorServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if cancel, ok := loadCancelFunction(ctx); ok {
		cancel()
	}

	return next.Server(ctx).Close(ctx, conn)
}

func (m *monitorServer) monitorNetworkService(monitorCtx context.Context, conn *networkservice.Connection, factory begin.EventFactory) {
	logger := log.FromContext(monitorCtx).WithField("monitorServer", "Find")
	for ; monitorCtx.Err() == nil; time.Sleep(time.Millisecond * 100) {
		// nolint:govet
		stream, err := m.nsClient.Find(monitorCtx, &registry.NetworkServiceQuery{
			Watch: true,
			NetworkService: &registry.NetworkService{
				Name: conn.GetNetworkService(),
			},
		})
		if err != nil {
			logger.Errorf("an error happened during finding network service: %v", err.Error())
			continue
		}

		networkServiceCh := registry.ReadNetworkServiceChannel(stream)
		netsvcStreamIsAlive := true

		for netsvcStreamIsAlive && monitorCtx.Err() == nil {
			select {
			case <-monitorCtx.Done():
				return
			case netsvc, ok := <-networkServiceCh:
				if !ok {
					netsvcStreamIsAlive = false
					break
				}

				nseStream, err := m.nseClient.Find(monitorCtx, &registry.NetworkServiceEndpointQuery{
					NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
						Name: conn.GetNetworkServiceEndpointName(),
					},
				})
				if err != nil {
					logger.Errorf("an error happened during finding nse: %v", err.Error())
					break
				}

				nses := registry.ReadNetworkServiceEndpointList(nseStream)

				if len(nses) == 0 {
					continue
				}

				if len(matchutils.MatchEndpoint(conn.GetLabels(), netsvc.GetNetworkService(), nses...)) == 0 {
					factory.Close()
					logger.Warnf("nse %v doesn't match with networkservice: %v", conn.GetNetworkServiceEndpointName(), conn.GetNetworkService())
					return
				}
			}
		}
	}
}
