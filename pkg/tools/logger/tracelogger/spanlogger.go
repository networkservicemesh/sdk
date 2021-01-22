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

package tracelogger

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
	"github.com/networkservicemesh/sdk/pkg/tools/logger"
)

const (
	maxStringLength int = 1000
	dotCount        int = 3
)

// spanlogger - provides a way to log via opentracing spans
type spanLogger struct {
	operation string
	span      opentracing.Span
	entries   map[interface{}]interface{}
}

func (s *spanLogger) Info(v ...interface{}) {
	s.log("info", v...)
}

func (s *spanLogger) Infof(format string, v ...interface{}) {
	s.logf("info", format, v...)
}

func (s *spanLogger) Warn(v ...interface{}) {
	s.log("warn", v...)
}

func (s *spanLogger) Warnf(format string, v ...interface{}) {
	s.logf("warn", format, v...)
}

func (s *spanLogger) Error(v ...interface{}) {
	s.WithField("stacktrace", limitString(string(debug.Stack()))).(*spanLogger).log("error", v...)
}

func (s *spanLogger) Errorf(format string, v ...interface{}) {
	s.WithField("stacktrace", limitString(string(debug.Stack()))).(*spanLogger).logf("error", format, v...)
}

func (s *spanLogger) Fatal(v ...interface{}) {
	s.log("fatal", v...)
}

func (s *spanLogger) Fatalf(format string, v ...interface{}) {
	s.logf("fatal", format, v...)
}

func (s *spanLogger) Debug(v ...interface{}) {
	s.log("debug", v...)
}

func (s *spanLogger) Debugf(format string, v ...interface{}) {
	s.logf("debug", format, v...)
}

func (s *spanLogger) Trace(v ...interface{}) {
	s.log("trace", v...)
}

func (s *spanLogger) Tracef(format string, v ...interface{}) {
	s.logf("trace", format, v...)
}

func (s *spanLogger) Object(k, v interface{}) {
	if s.span != nil {
		if v != nil {
			msg := ""
			cc, err := json.Marshal(v)
			if err == nil {
				msg = string(cc)
			} else {
				msg = fmt.Sprint(v)
			}

			s.span.LogFields(log.Object(k.(string), limitString(msg)))
			for k, v := range s.entries {
				s.span.LogKV(k, v)
			}
		}
	}
}

func (s *spanLogger) WithField(key, value interface{}) logger.Logger {
	data := make(map[interface{}]interface{}, len(s.entries)+1)
	for k, v := range s.entries {
		data[k] = v
	}
	data[key] = value
	newlog := &spanLogger{
		span:      s.span,
		operation: s.operation,
		entries:   data,
	}
	return newlog
}

func (s *spanLogger) log(level string, v ...interface{}) {
	s.logf(level, "%s", fmt.Sprint(v...))
}

func (s *spanLogger) logf(level, format string, v ...interface{}) {
	if s.span != nil {
		if v != nil {
			s.span.LogFields(log.String("event", level), log.String("message", fmt.Sprintf(format, v...)))
			for k, v := range s.entries {
				s.span.LogKV(k, v)
			}
		}
	}
}

// newSpanLogger - creates a new spanLogger from context and operation
func newSpanLogger(ctx context.Context, operation string) (context.Context, logger.Logger, opentracing.Span, func()) {
	var span opentracing.Span
	if jaeger.IsOpentracingEnabled() {
		span, ctx = opentracing.StartSpanFromContext(ctx, operation)
	}
	newLog := &spanLogger{
		span:      span,
		operation: operation,
		entries:   make(map[interface{}]interface{}),
	}
	return logger.WithLog(ctx, newLog), newLog, span, func() { newLog.finish() }
}

// finish - closes spanLogger
func (s *spanLogger) finish() {
	if s.span != nil {
		s.span.Finish()
		s.span = nil
	}
}

func limitString(s string) string {
	if len(s) > maxStringLength {
		return s[maxStringLength-dotCount:] + strings.Repeat(".", dotCount)
	}
	return s
}
