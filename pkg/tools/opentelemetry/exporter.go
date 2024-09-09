// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
// Copyright (c) 2023 Nordix Foundation.
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

// Package opentelemetry provides a set of utilities for assisting with telemetry data
package opentelemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// InitSpanExporter - returns an instance of OpenTelemetry Span Exporter.
func InitSpanExporter(ctx context.Context, exporterURL string) trace.SpanExporter {
	if !IsEnabled() {
		return nil
	}

	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(exporterURL),
		otlptracegrpc.WithDialOption(grpc.WithBlock()))
	exporter, err := otlptrace.New(ctx, traceClient)
	if err != nil {
		log.FromContext(ctx).Fatal(err)
		return nil
	}

	return exporter
}

// InitOPTLMetricExporter - returns an instance of OpenTelemetry Metric Exporter.
func InitOPTLMetricExporter(ctx context.Context, exporterURL string, exportInterval time.Duration) sdkmetric.Reader {
	if !IsEnabled() {
		return nil
	}
	conn, err := grpc.DialContext(
		ctx,
		exporterURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.FromContext(ctx).Errorf("%v", err)
		return nil
	}
	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		log.FromContext(ctx).Errorf("%v", err)
		return nil
	}

	return sdkmetric.NewPeriodicReader(
		exporter,
		sdkmetric.WithInterval(exportInterval),
	)
}

// InitPrometheusMetricExporter - returns an instance of Prometheus Metric Exporter.
func InitPrometheusMetricExporter(ctx context.Context) sdkmetric.Reader {
	if !IsEnabled() {
		return nil
	}
	exporter, err := prometheus.New()
	if err != nil {
		log.FromContext(ctx).Errorf("%v", err)
		return nil
	}

	return exporter
}
