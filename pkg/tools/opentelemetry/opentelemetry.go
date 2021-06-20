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
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/metric/global"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	// InstrumentationName - denotes the library that provides the instrumentation
	InstrumentationName = "NSM"
)

type opentelemetry struct {
	io.Closer

	ctx context.Context

	controller     *controller.Controller
	tracerProvider *sdktrace.TracerProvider
	exporter       *otlp.Exporter
}

func (o *opentelemetry) Close() error {
	if o.controller != nil {
		if err := o.controller.Stop(o.ctx); err != nil {
			log.FromContext(o.ctx).Errorf("failed to shutdown controller: %v", err)
		}
	}
	if o.tracerProvider != nil {
		if err := o.tracerProvider.Shutdown(o.ctx); err != nil {
			log.FromContext(o.ctx).Errorf("failed to shutdown provider: %v", err)
		}
	}
	if o.exporter != nil {
		if err := o.controller.Stop(o.ctx); err != nil {
			log.FromContext(o.ctx).Errorf("failed to stop exporter: %v", err)
		}
	}
	return nil
}

// Init - creates opentelemetry trace and metrics providers
func Init(ctx context.Context, service, otelAddr string) io.Closer {
	opentel := &opentelemetry{}
	if !log.IsOpentelemetryEnabled() {
		return opentel
	}

	opentel.ctx = ctx

	driverOptions := []otlpgrpc.Option{
		otlpgrpc.WithInsecure(),
	}
	if otelAddr != "" {
		driverOptions = append(driverOptions, otlpgrpc.WithEndpoint(otelAddr))
	}
	metricsDriver := otlpgrpc.NewDriver(driverOptions...)
	tracesDriver := otlpgrpc.NewDriver(driverOptions...)
	driver := otlp.NewSplitDriver(
		otlp.SplitConfig{
			ForMetrics: metricsDriver,
			ForTraces:  tracesDriver,
		},
	)

	exp, err := otlp.NewExporter(ctx, driver)
	if err != nil {
		log.FromContext(ctx).Errorf("%v", err)
		return opentel
	}
	opentel.exporter = exp

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(service),
		),
	)
	if err != nil {
		log.FromContext(ctx).Errorf("%v", err)
		return opentel
	}

	/* Create tracer provider */
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(
			exp,
			// add following two options to ensure flush
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(10),
		),
	)
	otel.SetTracerProvider(tracerProvider)
	opentel.tracerProvider = tracerProvider

	/* Create meter provider */
	cont := controller.New(
		processor.New(
			simple.NewWithExactDistribution(),
			exp,
		),
		controller.WithExporter(exp),
	)
	global.SetMeterProvider(cont.MeterProvider())

	if err := cont.Start(ctx); err != nil {
		log.FromContext(ctx).Errorf("%v", err)
		return opentel
	}
	opentel.controller = cont

	return opentel
}
