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

package roundrobin

import (
	"context"
	"net/url"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type selectEndpointServer struct {
	selector *roundRobinSelector
}

// NewServer - provides a NetworkServiceServer chain element that round robins among candidates provided by
// discover.Candidate(ctx) in the context.
func NewServer() networkservice.NetworkServiceServer {
	return &selectEndpointServer{
		selector: newRoundRobinSelector(),
	}
}

func (s *selectEndpointServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if clienturlctx.ClientURL(ctx) != nil {
		return next.Server(ctx).Request(ctx, request)
	}
	candidates := discover.Candidates(ctx)

	for i := 0; i < len(candidates.Endpoints); i++ {
		endpoint := s.selector.selectEndpoint(candidates.NetworkService, candidates.Endpoints)
		if endpoint == nil {
			return nil, errors.Errorf("failed to find endpoint for Network Service: %v %v", candidates.NetworkService, candidates.Endpoints)
		}
		u, err := url.Parse(endpoint.Url)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		ctx = clienturlctx.WithClientURL(ctx, u)
		request.GetConnection().NetworkServiceEndpointName = endpoint.Name
		resp, err := next.Server(ctx).Request(ctx, request)
		if err == nil {
			return resp, err
		}
	}
	return nil, errors.Errorf("all candidates %#v fail", candidates)
}

func (s *selectEndpointServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if clienturlctx.ClientURL(ctx) != nil {
		return next.Server(ctx).Close(ctx, conn)
	}
	return nil, errors.Errorf("passed incorrect connection: %+v", conn)
}
