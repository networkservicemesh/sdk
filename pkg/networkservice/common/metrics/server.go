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

// Package metrics provides a chain element that sends metrics to collector
package metrics

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/opentelemetry/meterhelper"
)

type metricServer struct {
	helpers map[string]meterhelper.MeterHelper
}

// NewServer returns a new metric server chain element
func NewServer() networkservice.NetworkServiceServer {
	return &metricServer{
		helpers: make(map[string]meterhelper.MeterHelper),
	}
}

func (t *metricServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	t.writeMetrics(ctx, conn.GetPath())
	return conn, nil
}

func (t *metricServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, err := next.Server(ctx).Close(ctx, conn)
	if err != nil {
		return nil, err
	}

	t.writeMetrics(ctx, conn.GetPath())
	return &empty.Empty{}, nil
}

func (t *metricServer) writeMetrics(ctx context.Context, path *networkservice.Path) {
	if path != nil {
		for _, pathSegment := range path.GetPathSegments() {
			if pathSegment.Metrics == nil {
				continue
			}
			_, ok := t.helpers[pathSegment.Id]
			if !ok {
				t.helpers[pathSegment.Id] = meterhelper.NewMeterHelper(pathSegment.Name, path.GetPathSegments()[0].Id)
			}
			t.helpers[pathSegment.Id].WriteMetrics(ctx, pathSegment.Metrics)
		}
	}
}
