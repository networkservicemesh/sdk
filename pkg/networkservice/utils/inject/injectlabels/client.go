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

// Package injectlabels sets connection labels
package injectlabels

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type injectLabelsClient struct {
	labels map[string]string
}

func (s *injectLabelsClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (conn *networkservice.Connection, err error) {
	oldConnLabels := request.Connection.Labels
	oldNetworkServiceEndpointName := request.Connection.NetworkServiceEndpointName

	request.Connection.Labels = s.labels
	request.Connection.NetworkServiceEndpointName = ""

	conn, err = next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	conn.Labels = oldConnLabels
	conn.NetworkServiceEndpointName = oldNetworkServiceEndpointName

	return conn, nil
}

func (s *injectLabelsClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

// NewClient creates new instance of NetworkServiceClient chain element, which inject labels into connection
func NewClient(labels map[string]string) networkservice.NetworkServiceClient {
	return &injectLabelsClient{
		labels: labels,
	}
}
