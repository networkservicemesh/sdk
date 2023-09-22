// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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
	"io"
	"os"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	telemetryEnv = "TELEMETRY"
)

// IsEnabled returns true if opentelemetry enabled
func IsEnabled() bool {
	if v, err := strconv.ParseBool(os.Getenv(telemetryEnv)); err == nil {
		return v
	}
	return false
}

type opentelemetry struct {
	io.Closer

	ctx context.Context
	/* Traces */
	tracerProvider *sdktrace.TracerProvider
	/* Metrics */
	metricController *sdkmetric.MeterProvider
}

func (o *opentelemetry) Close() error {
	if o.tracerProvider != nil {
		if err := o.tracerProvider.Shutdown(o.ctx); err != nil {
			log.FromContext(o.ctx).Errorf("failed to shutdown provider: %v", err)
		}
	}
	if o.metricController != nil {
		if err := o.metricController.Shutdown(o.ctx); err != nil {
			log.FromContext(o.ctx).Errorf("failed to shutdown controller: %v", err)
		}
	}
	return nil
}

// Init - creates opentelemetry tracer and meter providers
func Init(ctx context.Context, spanExporter sdktrace.SpanExporter, metricReader sdkmetric.Reader, service string) io.Closer {
	o := &opentelemetry{
		ctx: ctx,
	}
	if !IsEnabled() {
		return o
	}

	// Create resources
	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(service),
		),
	)
	if err != nil {
		log.FromContext(ctx).Errorf("%v", err)
		return o
	}

	// Create trace provider
	if spanExporter != nil {
		// Register the trace exporter with a TracerProvider, using a batch
		// span processor to aggregate spans before export.
		bsp := sdktrace.NewBatchSpanProcessor(spanExporter)
		tracerProvider := sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithResource(res),
			sdktrace.WithSpanProcessor(bsp),
		)

		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}))
		otel.SetTracerProvider(tracerProvider)
		o.tracerProvider = tracerProvider
	}

	// Create meter provider
	if metricReader != nil {
		meterProvider := sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(metricReader),
		)

		otel.SetMeterProvider(meterProvider)
		o.metricController = meterProvider
	}

	return o
}
