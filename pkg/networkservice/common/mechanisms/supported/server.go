// Copyright (c) 2021 Cisco and/or its affiliates.
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

package supported

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type supportedMechanismServer struct {
	supportedMechanismTypes []string
}

// NewServer - chain element to check that we have support for the mechanisms requested
func NewServer(options ...Option) networkservice.NetworkServiceServer {
	o := &supportedOptions{}
	for _, opt := range options {
		opt(o)
	}
	return &supportedMechanismServer{
		supportedMechanismTypes: o.supportedMechanismTypes,
	}
}

func (s *supportedMechanismServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// If we have a specified Mechanism... we really need to support it
	if request.GetConnection().GetMechanism() != nil {
		for _, supportedMechanismType := range s.supportedMechanismTypes {
			if request.GetConnection().GetMechanism().GetType() == supportedMechanismType {
				return next.Server(ctx).Request(ctx, request)
			}
		}
		return nil, errors.Errorf("Request.Connection.Mechanism.Type (%q) unsupported. Supported types are %q", request.GetConnection().GetMechanism().GetType(), s.supportedMechanismTypes)
	}

	// If we don't have any of the mechanism preferences fall through to return an error
	mechanismPreferenceTypes := []string{}
	for _, mechanism := range request.GetMechanismPreferences() {
		for _, supportedMechanismType := range s.supportedMechanismTypes {
			if mechanism.GetType() == supportedMechanismType {
				return next.Server(ctx).Request(ctx, request)
			}
		}
		mechanismPreferenceTypes = append(mechanismPreferenceTypes, mechanism.GetType())
	}
	return nil, errors.Errorf("All Request.MechanismPreferences[*].Types (%q) unsupported. Supported types are %q", mechanismPreferenceTypes, s.supportedMechanismTypes)
}

func (s *supportedMechanismServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, connection)
}
