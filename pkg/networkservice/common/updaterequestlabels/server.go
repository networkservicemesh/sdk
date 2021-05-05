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

// Package updaterequestlabels provides a networkservice chain element that updates Connection.Labels using discovery.Candidates
package updaterequestlabels

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type updateRequestLabelsServer struct{}

// NewServer - create chain element that will update Connection.Labels according to information from discover.Candidates
func NewServer() networkservice.NetworkServiceServer {
	return &updateRequestLabelsServer{}
}

func (s *updateRequestLabelsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	originalLabels := s.updateLabels(ctx, request.Connection)
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	conn.Labels = originalLabels
	return conn, nil
}

func (s *updateRequestLabelsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_ = s.updateLabels(ctx, conn)
	return next.Server(ctx).Close(ctx, conn)
}

// updateLabels - updates labels in conn, return original value of conn.Labels.
// If context doesn't contain info for network service specified in conn, does nothing.
func (s *updateRequestLabelsServer) updateLabels(ctx context.Context, conn *networkservice.Connection) map[string]string {
	originalLabels := conn.Labels
	labels, err := s.getLabelsForEndpoint(ctx, conn.NetworkServiceEndpointName)
	if err == nil {
		conn.Labels = labels
	}
	return originalLabels
}

// getLabelsForEndpoint - searches for labels linked to specified endpointName in context
func (s *updateRequestLabelsServer) getLabelsForEndpoint(ctx context.Context, endpointName string) (map[string]string, error) {
	candidates := discover.Candidates(ctx)
	if candidates == nil {
		return nil, errors.New("candidates are not found")
	}
	if candidates.DestLabels == nil {
		return nil, errors.New("candidates.DestLabels are empty")
	}

	idx := -1
	for i, nse := range candidates.Endpoints {
		if endpointName == nse.Name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, errors.New("endpoint with specified name is not found in candidates")
	}
	return candidates.DestLabels[idx], nil
}
