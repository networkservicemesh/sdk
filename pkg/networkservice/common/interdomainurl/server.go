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

// Package interdomainurl provides chain element to putting remote NSMgr URL into context.
package interdomainurl

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

type interdomainURLServer struct{}

func (i *interdomainURLServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	nseName := request.GetConnection().GetNetworkServiceEndpointName()
	if nseName == "" {
		return nil, errors.New("NSE is not selected")
	}
	remoteURL := interdomain.Domain(nseName)
	u, err := url.Parse(remoteURL)
	if err != nil {
		return nil, errors.Wrap(err, "selected NSE has wrong name. Make sure that proxy-registry has handled NSE")
	}
	request.GetConnection().NetworkServiceEndpointName = interdomain.Target(nseName)
	return next.Server(ctx).Request(clienturlctx.WithClientURL(ctx, u), request)
}

func (i *interdomainURLServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	nseName := connection.GetNetworkServiceEndpointName()
	if nseName == "" {
		return nil, errors.New("NSE is not selected")
	}
	remoteURL := interdomain.Domain(nseName)
	u, err := url.Parse(remoteURL)
	if err != nil {
		return nil, errors.Wrap(err, "selected NSE has wrong name. Make sure that proxy-registry has handled NSE")
	}
	connection.NetworkServiceEndpointName = interdomain.Target(nseName)
	return next.Server(ctx).Close(clienturlctx.WithClientURL(ctx, u), connection)
}

// NewServer creates new interdomainurl chain element
func NewServer() networkservice.NetworkServiceServer {
	return &interdomainURLServer{}
}
