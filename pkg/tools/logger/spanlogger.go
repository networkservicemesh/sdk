// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package logger

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
)

type spanLogger struct {
	operation string
	span      opentracing.Span
	entries   map[interface{}]interface{}
}

func (s *spanLogger) Info(v ...interface{}) {
	s.log("info", v)
}

func (s *spanLogger) Infof(format string, v ...interface{}) {
	s.logf("info", format, v...)
}

func (s *spanLogger) Warn(v ...interface{}) {
	s.log("warn", v)
}

func (s *spanLogger) Warnf(format string, v ...interface{}) {
	s.logf("warn", format, v...)
}

func (s *spanLogger) Error(v ...interface{}) {
	s.WithField("stacktrace", limitString(string(debug.Stack()))).(*spanLogger).log("error", v)
}

func (s *spanLogger) Errorf(format string, v ...interface{}) {
	s.WithField("stacktrace", limitString(string(debug.Stack()))).(*spanLogger).logf("error", format, v...)
}

func (s *spanLogger) Fatal(v ...interface{}) {
	s.log("fatal", v)
}

func (s *spanLogger) Fatalf(format string, v ...interface{}) {
	s.logf("fatal", format, v...)
}

func (s *spanLogger) WithField(key, value interface{}) Logger {
	data := make(map[interface{}]interface{}, len(s.entries)+1)
	for k, v := range s.entries {
		data[k] = v
	}
	data[key] = value
	logger := &spanLogger{
		span:      s.span,
		operation: s.operation,
		entries:   data,
	}
	return logger
}

func (s *spanLogger) log(level string, v ...interface{}) {
	s.logf(level, format(v), v)
}

func (s *spanLogger) logf(level, format string, v ...interface{}) {
	if s.span != nil {
		if v != nil {
			msg := limitString(fmt.Sprintf(format, v...))
			s.span.LogFields(log.String("event", level), log.String("message", msg))
			for k, v := range s.entries {
				s.span.LogKV(k, v)
			}
		}
	}
}

// NewSpan - creates a new spanLogger from context, operation and span
func NewSpan(ctx context.Context, operation string) (Logger, context.Context) {
	var span opentracing.Span
	if jaeger.IsOpentracingEnabled() {
		span, ctx = opentracing.StartSpanFromContext(ctx, operation)
	}
	logger := &spanLogger{
		span:      span,
		operation: operation,
		entries:   make(map[interface{}]interface{}),
	}
	return logger, ctx
}

// Close - closes spanLogger
func (s *spanLogger) Finish() {
	if s.span != nil {
		s.span.Finish()
		s.span = nil
	}
}

// Span - returns opentracing span for this logger
func (s *spanLogger) Span() opentracing.Span {
	return s.span
}
