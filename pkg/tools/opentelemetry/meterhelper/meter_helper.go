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

// Package meterhelper provides a set of utilities to assist in working with opentelemetry metrics
package meterhelper

import (
	"context"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/opentelemetry"
)

// MeterHelper - wrap opentelemetry Meter to simplify workflow
type MeterHelper interface {
	WriteMetrics(ctx context.Context, metrics map[string]string)
}

type meterHelper struct {
	prefix      string
	connLabel   attribute.KeyValue
	meter       metric.Meter
	recorderMap map[string]metric.Int64ValueRecorder
}

// NewMeterHelper - constructs a meter helper from segmentName and connectionID.
func NewMeterHelper(segmentName, connectionID string) MeterHelper {
	meter := global.Meter(opentelemetry.InstrumentationName)
	return &meterHelper{
		prefix:      segmentName + "_",
		connLabel:   attribute.String("connection", connectionID),
		meter:       meter,
		recorderMap: make(map[string]metric.Int64ValueRecorder),
	}
}

func (m *meterHelper) WriteMetrics(ctx context.Context, metrics map[string]string) {
	if metrics == nil || !log.IsOpentelemetryEnabled() {
		return
	}

	for metricName, metricValue := range metrics {
		/* Works with integers only */
		recVal, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			continue
		}
		_, ok := m.recorderMap[metricName]
		if !ok {
			m.recorderMap[metricName] = metric.Must(m.meter).NewInt64ValueRecorder(
				m.prefix + metricName,
			)
		}
		m.recorderMap[metricName].Record(ctx, recVal, m.connLabel)
	}
}
