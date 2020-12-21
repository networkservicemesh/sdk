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

// Package tracelogger provides wrapper for logrus logger
// which is consistent with Logger interface
// and sends messages containing tracing information
package tracelogger

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
	"github.com/networkservicemesh/sdk/pkg/tools/logger"
	"github.com/networkservicemesh/sdk/pkg/tools/logruslogger"
)

type traceLogger struct {
	operation string
	span      opentracing.Span
	info      *traceCtxInfo
	entry     *logrus.Entry
}

func (s *traceLogger) Info(v ...interface{}) {
	s.log(v)
}

func (s *traceLogger) Infof(format string, v ...interface{}) {
	s.logf(format, v...)
}

func (s *traceLogger) Warn(v ...interface{}) {
	s.log(v)
}

func (s *traceLogger) Warnf(format string, v ...interface{}) {
	s.logf(format, v...)
}

func (s *traceLogger) Error(v ...interface{}) {
	s.log(v)
}

func (s *traceLogger) Errorf(format string, v ...interface{}) {
	s.logf(format, v...)
}

func (s *traceLogger) Fatal(v ...interface{}) {
	s.log(v)
}

func (s *traceLogger) Fatalf(format string, v ...interface{}) {
	s.logf(format, v...)
}

// New - returns a new traceLogger from context and span with given operation name
func New(ctx context.Context, operation string, span opentracing.Span) (logger.Logger, context.Context) {
	var fields map[string]string = nil
	if value, ok := ctx.Value(logruslogger.CtxKeyLogEntry).(map[string]string); ok {
		fields = value
	}
	entry := logrus.WithTime(time.Now()).WithContext(ctx)
	for k, v := range fields {
		entry = entry.WithField(k, v)
	}
	if jaeger.IsOpentracingEnabled() && span == nil {
		span, ctx = opentracing.StartSpanFromContext(ctx, operation)
	}
	var info *traceCtxInfo
	ctx, info = withTraceInfo(ctx)
	localTraceInfo.Store(info.id, info)
	log := &traceLogger{
		span:      span,
		info:      info,
		operation: operation,
		entry:     entry,
	}
	ctx = logger.WithLog(ctx, log)
	log.printStart(operation)
	return log, ctx
}

func (s *traceLogger) WithField(key, value interface{}) logger.Logger {
	entry := s.entry
	entry = entry.WithFields(logrus.Fields{key.(string): value})
	log := &traceLogger{span: s.span, entry: entry, info: s.info}
	return log
}

func (s *traceLogger) log(v ...interface{}) {
	s.logf(format(v), v)
}

func (s *traceLogger) logf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	incInfo := s.info.incInfo()
	s.entry.Tracef("%v %s %v%v", incInfo, strings.Repeat(" ", s.info.level), msg, s.getSpan())
}

func (s *traceLogger) getSpan() string {
	spanStr := fmt.Sprintf("%v", s.span)
	if len(spanStr) > 0 && spanStr != "{}" && s.span != nil {
		return fmt.Sprintf(" span=%v", spanStr)
	}
	return ""
}

func (s *traceLogger) printStart(operation string) {
	prefix := strings.Repeat(" ", s.info.level)
	incInfo := s.info.incInfo()
	s.entry.Tracef("%v%s⎆ %v()%v", incInfo, prefix, operation, s.getSpan())
}
