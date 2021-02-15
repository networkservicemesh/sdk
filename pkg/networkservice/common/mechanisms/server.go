// Copyright (c) 2020-2021 Cisco and/or its affiliates.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

// Package mechanisms provides a simple shim to allow the attempt to select a mechanism based on the MechanismPreference
// expressed in the Request.
package mechanisms

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
)

type mechanismsServer struct {
	mechanisms map[string]networkservice.NetworkServiceServer // key is Mechanism.Type
}

// NewServer - returns new NetworkServiceServer chain element that will attempt to meet the request.MechanismPreferences using
//             the provided maps of MechanismType to NetworkServiceServer
//             - mechanisms
//                   key:    mechanismType
//                   value:  NetworkServiceServer that only handles the work for the specified mechanismType
//                           Note: Supplied NetworkServiceServer elements should not call next.Server(ctx).{Request,Close} themselves
func NewServer(mechanisms map[string]networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	rv := &mechanismsServer{
		mechanisms: make(map[string]networkservice.NetworkServiceServer),
	}
	for mechanismType, server := range mechanisms {
		// We wrap in a chain here to make sure that if the 'server' is calling next.Server(ctx) it doesn't
		// skips past returning here.
		rv.mechanisms[mechanismType] = chain.NewNetworkServiceServer(server)
	}
	return rv
}

func (ms *mechanismsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetMechanism() != nil {
		srv, ok := ms.mechanisms[request.GetConnection().GetMechanism().GetType()]
		if ok {
			return srv.Request(ctx, request)
		}
		return nil, errUnsupportedMech
	}
	var err = errCannotSupportMech
	for _, mechanism := range request.GetMechanismPreferences() {
		srv, ok := ms.mechanisms[mechanism.GetType()]
		if ok {
			req := request.Clone()
			req.GetConnection().Mechanism = mechanism
			var resp *networkservice.Connection
			resp, respErr := srv.Request(ctx, req)
			if respErr == nil {
				return resp, nil
			}
			err = errors.Wrap(err, respErr.Error())
		}
	}
	return nil, err
}

func (ms *mechanismsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	srv, ok := ms.mechanisms[conn.GetMechanism().GetType()]
	if ok {
		return srv.Close(ctx, conn)
	}
	return nil, errCannotSupportMech
}
