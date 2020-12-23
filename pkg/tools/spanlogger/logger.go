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

// Package spanlogger provides a way to log via opentracing spans
// and is consistent with logger.Logger interface
package spanlogger

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
	"github.com/networkservicemesh/sdk/pkg/tools/logger"
)

type spanLogger struct {
	mutex     *sync.Mutex
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

func (s *spanLogger) WithField(key, value interface{}) logger.Logger {
	data := make(map[interface{}]interface{}, len(s.entries)+1)
	for k, v := range s.entries {
		data[k] = v
	}
	data[key] = value
	s.mutex.Lock()
	newlog := &spanLogger{
		mutex:     s.mutex,
		span:      s.span,
		operation: s.operation,
		entries:   data,
	}
	s.mutex.Unlock()
	return newlog
}

func (s *spanLogger) log(level string, v ...interface{}) {
	s.logf(level, format(v), v)
}

func (s *spanLogger) logf(level, format string, v ...interface{}) {
	s.mutex.Lock()
	if s.span != nil {
		if v != nil {
			msg := limitString(fmt.Sprintf(format, v...))
			s.span.LogFields(log.String("event", level), log.String("message", msg))
			for k, v := range s.entries {
				s.span.LogKV(k, v)
			}
		}
	}
	s.mutex.Unlock()
}

// New - creates a new spanLogger from context, operation and span
func New(ctx context.Context, operation string) (logger.Logger, context.Context, opentracing.Span, func()) {
	var span opentracing.Span
	if jaeger.IsOpentracingEnabled() {
		span, ctx = opentracing.StartSpanFromContext(ctx, operation)
	}
	newLog := &spanLogger{
		mutex:     &sync.Mutex{},
		span:      span,
		operation: operation,
		entries:   make(map[interface{}]interface{}),
	}
	return newLog, ctx, span, func() { newLog.Finish() }
}

// Close - closes spanLogger
func (s *spanLogger) Finish() {
	s.mutex.Lock()
	if s.span != nil {
		s.span.Finish()
		s.span = nil
	}
	s.mutex.Unlock()
}
