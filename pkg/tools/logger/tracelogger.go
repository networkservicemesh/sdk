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
	"strings"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

type traceLogger struct {
	operation string
	span      opentracing.Span
	ctx       context.Context
	info      *traceCtxInfo
	entry     *logrus.Entry
}

func (s *traceLogger) Info(v ...interface{}) {
	s.log(INFO, v)
}

func (s *traceLogger) Infof(format string, v ...interface{}) {
	s.logf(INFO, format, v...)
}

func (s *traceLogger) Warn(v ...interface{}) {
	s.log(WARN, v)
}

func (s *traceLogger) Warnf(format string, v ...interface{}) {
	s.logf(WARN, format, v...)
}

func (s *traceLogger) Error(v ...interface{}) {
	s.log(ERROR, v)
}

func (s *traceLogger) Errorf(format string, v ...interface{}) {
	s.logf(ERROR, format, v...)
}

func (s *traceLogger) Fatal(v ...interface{}) {
	s.log(FATAL, v)
}

func (s *traceLogger) Fatalf(format string, v ...interface{}) {
	s.logf(FATAL, format, v...)
}

func getEntry(ctx context.Context) *logrus.Entry {
	if value, ok := ctx.Value(CTXKEY_LOGRUS_ENTRY).(*logrus.Entry); ok {
		return value
	}
	return logrus.WithTime(time.Now())
}

func NewTrace(ctx context.Context, operation string, span opentracing.Span) Logger {
	if jaeger.IsOpentracingEnabled() && span == nil {
		span, ctx = opentracing.StartSpanFromContext(ctx, operation)
	}
	var info *traceCtxInfo
	ctx, info = withTraceInfo(ctx)
	localTraceInfo.Store(info.id, info)
	logger := &traceLogger{span: span, info: info}
	ctx = context.WithValue(ctx, CTXKEY_LOGGER, logger)
	logger.entry = getEntry(ctx).WithContext(ctx)
	logger.ctx = ctx
	logger.printStart(operation)
	return logger
}

func (s *traceLogger) WithField(key, value interface{}) Logger {
	entry := s.entry
	entry = entry.WithFields(logrus.Fields{key.(string): value})
	logger := &traceLogger{span: s.span, entry: entry}
	ctx := context.WithValue(entry.Context, CTXKEY_LOGRUS_ENTRY, entry)
	ctx = context.WithValue(ctx, CTXKEY_LOGGER, logger)
	entry.Context = ctx
	return logger
}

func (s *traceLogger) log(level loggerLevel, v ...interface{}) {
	s.logf(level, format(v), v)
}

func (s *traceLogger) logf(level loggerLevel, format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	s.entry.Infof("%v %s %s=%v%v", s.info.incInfo(), strings.Repeat(separator, s.info.level), levelName(level), msg, s.getSpan())
}

func (s *traceLogger) getSpan() string {
	spanStr := fmt.Sprintf("%v", s.span)
	if len(spanStr) > 0 && spanStr != "{}" && s.span != nil {
		return fmt.Sprintf(" span=%v", spanStr)
	}
	return ""
}

func (s *traceLogger) printStart(operation string) {
	prefix := strings.Repeat(startSeparator, s.info.level)
	s.entry.Infof("%v%s⎆ %v()%v", s.info.incInfo(), prefix, operation, s.getSpan())
}

//Returns context with logger in it
func (s *traceLogger) Context() context.Context {
	return s.ctx
}

//Returns opentracing span for this logger
func (s *traceLogger) Span() opentracing.Span {
	return s.span
}

//Returns operation name
func (s *traceLogger) Operation() string {
	return s.operation
}
