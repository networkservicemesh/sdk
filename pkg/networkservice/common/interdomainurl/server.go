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
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

type interdomainURLServer struct{}

// NewServer creates new interdomainurl chain element
func NewServer() networkservice.NetworkServiceServer {
	return &interdomainURLServer{}
}

func (i *interdomainURLServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	interDomainNSEName := request.GetConnection().GetNetworkServiceEndpointName()
	nseName, domainURL, err := parseInterDomainNSEName(interDomainNSEName)
	if err != nil {
		return nil, err
	}
	request.GetConnection().NetworkServiceEndpointName = nseName

	conn, err := next.Server(ctx).Request(clienturlctx.WithClientURL(ctx, domainURL), request)
	if err != nil {
		return nil, err
	}
	conn.NetworkServiceEndpointName = interDomainNSEName

	return conn, nil
}

func (i *interdomainURLServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	interDomainNSEName := conn.GetNetworkServiceEndpointName()
	nseName, domainURL, err := parseInterDomainNSEName(interDomainNSEName)
	if err != nil {
		return nil, err
	}
	conn.NetworkServiceEndpointName = nseName

	return next.Server(ctx).Close(clienturlctx.WithClientURL(ctx, domainURL), conn)
}

func parseInterDomainNSEName(interDomainNSEName string) (string, *url.URL, error) {
	if interDomainNSEName == "" {
		return "", nil, errors.New("NSE is not selected")
	}

	remoteURL := interdomain.Domain(interDomainNSEName)
	u, err := url.Parse(remoteURL)
	if err != nil {
		return "", nil, errors.Wrap(err, "selected NSE has wrong name. Make sure that proxy-registry has handled NSE")
	}

	return interdomain.Target(interDomainNSEName), u, nil
}
