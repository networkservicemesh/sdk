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

package spanlogger

import (
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	opentelemetry "go.opentelemetry.io/otel/trace"

	opentelemetrynsm "github.com/networkservicemesh/sdk/pkg/tools/opentelemetry"
)

// Span - unified interface for opentracing/opentelemetry spans
type Span interface {
	Log(level, format string, v ...interface{})
	LogObject(k, v interface{})
	WithField(k, v interface{}) Span
	Finish()

	ToString() string
}

// Opentracing span
type otSpan struct {
	span opentracing.Span
}

func (otsp *otSpan) Log(level, format string, v ...interface{}) {
	otsp.span.LogFields(
		opentracinglog.String("event", level),
		opentracinglog.String("message", fmt.Sprintf(format, v...)),
	)
}

func (otsp *otSpan) LogObject(k, v interface{}) {
	otsp.span.LogFields(opentracinglog.Object(k.(string), v))
}

func (otsp *otSpan) WithField(k, v interface{}) Span {
	otsp.span = otsp.span.SetTag(k.(string), v)
	return otsp
}

func (otsp *otSpan) ToString() string {
	if spanStr := fmt.Sprintf("%v", otsp.span); spanStr != "{}" {
		return spanStr
	}
	return ""
}

func (otsp *otSpan) Finish() {
	otsp.span.Finish()
}

func newOTSpan(ctx context.Context, operationName string, additionalFields map[string]interface{}) (c context.Context, s Span) {
	span, ctx := opentracing.StartSpanFromContext(ctx, operationName)
	for k, v := range additionalFields {
		span = span.SetTag(k, v)
	}
	return ctx, &otSpan{span: span}
}

// Opentelemetry span
type otelSpan struct {
	span opentelemetry.Span
}

func (otelsp *otelSpan) Log(level, format string, v ...interface{}) {
	otelsp.span.AddEvent(
		"",
		opentelemetry.WithAttributes([]attribute.KeyValue{
			attribute.String("event", level),
			attribute.String("message", fmt.Sprintf(format, v...)),
		}...),
	)
}

func (otelsp *otelSpan) LogObject(k, v interface{}) {
	otelsp.span.AddEvent(
		"",
		opentelemetry.WithAttributes([]attribute.KeyValue{
			attribute.String(fmt.Sprintf("%v", k), fmt.Sprintf("%v", v)),
		}...),
	)
}

func (otelsp *otelSpan) WithField(k, v interface{}) Span {
	otelsp.span.SetAttributes(attribute.Any(k.(string), v))
	return otelsp
}

func (otelsp *otelSpan) ToString() string {
	if spanID := otelsp.span.SpanContext().SpanID(); spanID.IsValid() {
		return spanID.String()
	}
	return ""
}

func (otelsp *otelSpan) Finish() {
	otelsp.span.End()
}

func newOTELSpan(ctx context.Context, operationName string, additionalFields map[string]interface{}) (c context.Context, s Span) {
	var add []attribute.KeyValue

	for k, v := range additionalFields {
		add = append(add, attribute.Any(k, v))
	}

	ctx, span := otel.Tracer(opentelemetrynsm.InstrumentationName).Start(ctx, operationName)
	span.SetAttributes(add...)

	return ctx, &otelSpan{span: span}
}
