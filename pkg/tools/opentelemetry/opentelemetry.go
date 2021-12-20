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

// Package opentelemetry provides a set of utilities for assisting with telemetry data
package opentelemetry

import (
	"context"
	"io"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric/global"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	// InstrumentationName - denotes the library that provides the instrumentation
	InstrumentationName = "NSM"

	// defaults denotes default collector address
	defaultAddr = "localhost"
	defaultPort = "4317"
)

type opentelemetry struct {
	io.Closer

	ctx context.Context
	/* Traces */
	tracerProvider *sdktrace.TracerProvider
	/* Metrics */
	metricController *controller.Controller
	metricExporter   *otlpmetric.Exporter
}

func (o *opentelemetry) Close() error {
	if o.tracerProvider != nil {
		if err := o.tracerProvider.Shutdown(o.ctx); err != nil {
			log.FromContext(o.ctx).Errorf("failed to shutdown provider: %v", err)
		}
	}
	if o.metricController != nil {
		if err := o.metricController.Stop(o.ctx); err != nil {
			log.FromContext(o.ctx).Errorf("failed to shutdown controller: %v", err)
		}
	}
	if o.metricExporter != nil {
		if err := o.metricExporter.Shutdown(o.ctx); err != nil {
			log.FromContext(o.ctx).Errorf("failed to stop exporter: %v", err)
		}
	}
	return nil
}

// Init - creates opentelemetry tracer and meter providers
func Init(ctx context.Context, collectorAddr, service string) io.Closer {
	o := &opentelemetry{
		ctx: ctx,
	}
	if !log.IsOpentelemetryEnabled() {
		return o
	}

	// Check the opentlemetry collector address
	if collectorAddr == "" {
		collectorAddr = defaultAddr + ":" + defaultPort
	} else if len(strings.Split(collectorAddr, ":")) == 1 {
		collectorAddr += ":" + defaultPort
	}

	// Create tracer provider
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
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(collectorAddr),
	)
	if err != nil {
		log.FromContext(ctx).Errorf("%v", err)
		return o
	}
	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)
	o.tracerProvider = tracerProvider

	// Create meter provider
	client := otlpmetricgrpc.NewClient(
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(collectorAddr),
	)
	metricExporter, err := otlpmetric.New(ctx, client)
	if err != nil {
		log.FromContext(ctx).Errorf("%v", err)
		return o
	}
	o.metricExporter = metricExporter

	metricController := controller.New(
		processor.NewFactory(
			simple.NewWithExactDistribution(),
			metricExporter,
		),
		controller.WithExporter(metricExporter),
		controller.WithCollectPeriod(2*time.Second),
	)

	if err := metricController.Start(ctx); err != nil {
		log.FromContext(ctx).Errorf("%v", err)
		return o
	}
	global.SetMeterProvider(metricController)
	o.metricController = metricController

	return o
}
