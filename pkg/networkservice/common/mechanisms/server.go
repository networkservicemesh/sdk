// Copyright (c) 2020 Cisco and/or its affiliates.
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

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
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
	return &mechanismsServer{mechanisms: mechanisms}
}

func (m mechanismsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetMechanism() != nil {
		srv, ok := m.mechanisms[request.GetConnection().GetMechanism().GetType()]
		if ok {
			conn, err := srv.Request(ctx, request)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			request.Connection = conn
			return next.Server(ctx).Request(ctx, request)
		}
		return nil, errors.Errorf("Unsupported Mechanism: %+v", request.GetConnection().GetMechanism())
	}
	for _, mechanism := range request.GetMechanismPreferences() {
		srv, ok := m.mechanisms[mechanism.GetType()]
		if ok {
			req := request.Clone()
			req.GetConnection().Mechanism = mechanism
			conn, err := srv.Request(ctx, req)
			if err == nil {
				req.Connection = conn
				return next.Server(ctx).Request(ctx, req)
			}
		}
	}
	return nil, errors.Errorf("Cannot support any of the requested Mechanisms: %+v", request.GetMechanismPreferences())
}

func (m mechanismsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	srv, ok := m.mechanisms[conn.GetMechanism().GetType()]
	if ok {
		_, err := srv.Close(ctx, conn)
		if err == nil {
			return next.Server(ctx).Close(ctx, conn)
		}
	}
	return nil, errors.Errorf("Cannot support any of the requested Mechanism: %+v", conn.GetMechanism())
}
