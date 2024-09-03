// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022 Cisco and/or its affiliates.
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

// Package replacelabels sets connection labels
package replacelabels

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type replaceLabelsClient struct {
	labels map[string]string
}

// NewClient creates new instance of NetworkServiceClient chain element, which replaces labels in the connection.
func NewClient(labels map[string]string) networkservice.NetworkServiceClient {
	return &replaceLabelsClient{
		labels: labels,
	}
}

func (s *replaceLabelsClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (conn *networkservice.Connection, err error) {
	prevConnLabels := request.GetConnection().GetLabels()
	request.Connection.Labels = s.labels

	conn, err = next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	conn.Labels = prevConnLabels

	return conn, nil
}

func (s *replaceLabelsClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	conn.Labels = s.labels
	return next.Client(ctx).Close(ctx, conn, opts...)
}
