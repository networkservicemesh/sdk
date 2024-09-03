// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
// Copyright (c) 2023 Nordix Foundation.
// Copyright (c) 2024 Cisco and/or its affiliates.
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
	"fmt"
	"strconv"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/opentelemetry"
)

type metricServer struct {
	meter metric.Meter
}

// NewServer returns a new metric server chain element.
func NewServer() networkservice.NetworkServiceServer {
	res := &metricServer{}
	if opentelemetry.IsEnabled() {
		res.meter = otel.Meter("")
	}
	return res
}

func (t *metricServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if opentelemetry.IsEnabled() {
		t.writeMetrics(ctx, conn.GetPath())
	}
	return conn, nil
}

func (t *metricServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, err := next.Server(ctx).Close(ctx, conn)
	if err != nil {
		return nil, err
	}

	if opentelemetry.IsEnabled() {
		t.writeMetrics(ctx, conn.GetPath())
	}
	return &empty.Empty{}, nil
}

func (t *metricServer) writeMetrics(ctx context.Context, path *networkservice.Path) {
	if path != nil {
		for _, pathSegment := range path.GetPathSegments() {
			if pathSegment.Metrics == nil {
				continue
			}

			k := metricsData{
				counter: make(map[string]metric.Int64Counter),
			}
			metrics, _ := loadOrStore(ctx, &k)
			for metricName, metricValue := range pathSegment.GetMetrics() {
				/* Works with integers only */
				recVal, err := strconv.ParseInt(metricValue, 10, 64)
				if err != nil {
					continue
				}

				counterName := fmt.Sprintf("%s_%s", pathSegment.GetName(), metricName)
				_, ok := metrics.counter[metricName]
				if !ok {
					var counter metric.Int64Counter

					counter, err = t.meter.Int64Counter(counterName)
					if err != nil {
						continue
					}
					metrics.counter[metricName] = counter
				}

				previousValueKey := fmt.Sprintf(
					"%s.%s",
					counterName,
					path.GetPathSegments()[0].GetId(),
				)
				var previousValueInt int64
				previousValue, ok := metrics.previous.Load(previousValueKey)
				if ok {
					previousValueInt, _ = strconv.ParseInt(previousValue.(string), 10, 64)
				}

				metrics.counter[metricName].Add(
					ctx,
					recVal-previousValueInt,
					metric.WithAttributes(attribute.String("connection", path.GetPathSegments()[0].GetId())),
				)
				metrics.previous.Store(previousValueKey, metricValue)
			}
		}
	}
}
