// Copyright (c) 2020 Cisco Systems, Inc.
//
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

// Package jaeger provides a set of utilities for assisting with using jaeger
package jaeger

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/trace"
)

// InitExporter -  returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
func InitExporter(ctx context.Context, exporterURL string) trace.SpanExporter {
	if !log.IsOpentelemetryEnabled() {
		return nil
	}

	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(exporterURL)))
	go func() {
		<-ctx.Done()
		if err := exporter.Shutdown(context.Background()); err != nil {
			log.FromContext(ctx).Fatal(err)
		}
	}()

	if err != nil {
		log.FromContext(ctx).Fatal(err)
		return nil
	}

	return exporter
}
