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

package spanlogger

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	opentelemetry "go.opentelemetry.io/otel/trace"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// Span - unified interface for opentelemetry spans.
type Span interface {
	Log(level, format string, v ...interface{})
	LogObject(k, v interface{})
	WithField(k, v interface{}) Span
	Finish()

	ToString() string
}

// Opentelemetry span.
type otelSpan struct {
	rootSpan      opentelemetry.Span
	span          opentelemetry.Span
	operationName string
}

func (otelsp *otelSpan) Log(level, format string, v ...interface{}) {
	otelsp.span.AddEvent(
		otelsp.operationName,
		opentelemetry.WithAttributes([]attribute.KeyValue{
			attribute.String("event", level),
			attribute.String("message", fmt.Sprintf(format, v...)),
		}...),
	)
}

func (otelsp *otelSpan) LogObject(k, v interface{}) {
	otelsp.span.AddEvent(
		otelsp.operationName,
		opentelemetry.WithAttributes([]attribute.KeyValue{
			attribute.String(fmt.Sprintf("%v", k), fmt.Sprintf("%v", v)),
		}...),
	)
}

func (otelsp *otelSpan) WithField(k, v interface{}) Span {
	otelsp.span.SetAttributes(attribute.String(k.(string), fmt.Sprint(v)))
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
	if otelsp.rootSpan != nil {
		otelsp.rootSpan.End()
	}
}

func newOpentelemetrySpan(ctx context.Context, operationName, methodName string, additionalFields []*log.Field) (c context.Context, s Span) {
	var attributes []attribute.KeyValue

	for _, field := range additionalFields {
		attributes = append(attributes, attribute.String(field.Key(), fmt.Sprint(field.Val())))
	}

	// Check if the current span is active
	// If not (missing) - add a new root
	var rootSpan opentelemetry.Span
	if !opentelemetry.SpanFromContext(ctx).IsRecording() {
		var rootSpanName string
		for _, field := range additionalFields {
			rootSpanName += fmt.Sprintf("[%s: %s] ", field.Key(), fmt.Sprint(field.Val()))
		}
		rootSpanName += fmt.Sprintf("[%s: %s]", "method", methodName)
		ctx, rootSpan = otel.Tracer("").Start(ctx, rootSpanName, opentelemetry.WithNewRoot())
		rootSpan.SetAttributes(attributes...)
	}
	ctx, span := otel.Tracer("").Start(ctx, operationName)
	span.SetAttributes(attributes...)

	return ctx, &otelSpan{rootSpan: rootSpan, span: span, operationName: operationName}
}
