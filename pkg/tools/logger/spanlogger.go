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

	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

type spanLogger struct {
	operation string
	span      opentracing.Span
	ctx       context.Context
	//info      *traceCtxInfo
	//entryPool sync.Pool
	entries entriesType
	level   loggerLevel
}

func (s *spanLogger) Info(v ...interface{}) {
	s.log(INFO, v)
}

func (s *spanLogger) Infof(format string, v ...interface{}) {
	s.logf(INFO, format, v...)
}

func (s *spanLogger) Warn(v ...interface{}) {
	s.log(WARN, v)
}

func (s *spanLogger) Warnf(format string, v ...interface{}) {
	s.logf(WARN, format, v...)
}

func (s *spanLogger) Error(v ...interface{}) {
	s.log(ERROR, v)
}

func (s *spanLogger) Errorf(format string, v ...interface{}) {
	s.logf(ERROR, format, v...)
}

func (s *spanLogger) Fatal(v ...interface{}) {
	s.log(FATAL, v)
}

func (s *spanLogger) Fatalf(format string, v ...interface{}) {
	s.logf(FATAL, format, v...)
}

func (s *spanLogger) WithField(key, value interface{}) Logger {
	data := make(entriesType, len(s.entries)+1)
	for k, v := range s.entries {
		data[k] = v
	}
	data[key] = value
	logger := &spanLogger{
		ctx:       s.ctx,
		span:      s.span,
		operation: s.operation,
		level:     s.level,
		entries:   data,
	}
	logger.ctx = context.WithValue(s.ctx, CTXKEY_LOGGER, logger)
	return logger
}

func (s *spanLogger) log(level loggerLevel, v ...interface{}) {
	s.logf(level, format(v), v)
}

func (s *spanLogger) logf(level loggerLevel, format string, v ...interface{}) {
	if s.span != nil {
		if v != nil {
			msg := limitString(fmt.Sprintf(format, v...))
			s.span.LogFields(log.String("event", levelName(level)), log.String("message", msg))
			for k, v := range s.entries {
				s.span.LogKV(k, v)
			}
		}
	}
}

//Creates a new spanLogger from context, operation and span
func NewSpan(ctx context.Context, operation string) Logger {
	var span opentracing.Span
	if jaeger.IsOpentracingEnabled() {
		span, ctx = opentracing.StartSpanFromContext(ctx, operation)
	}
	logger := &spanLogger{
		span:      span,
		operation: operation,
		entries:   make(entriesType),
	}
	logger.ctx = context.WithValue(ctx, CTXKEY_LOGGER, logger)
	return logger
}

//Closes spanLogger
func (s *spanLogger) Close() {
	if s.span != nil {
		s.span.Finish()
		s.span = nil
	}
}

//Returns context with logger in it
func (s *spanLogger) Context() context.Context {
	return s.ctx
}

//Returns opentracing span for this logger
func (s *spanLogger) Span() opentracing.Span {
	return s.span
}

//Returns operation name
func (s *spanLogger) Operation() string {
	return s.operation
}
