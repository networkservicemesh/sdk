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

// Package logruslogger provides wrapper for logrus logger
// which is consistent with Logger interface
package logruslogger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"
)

type loggerKeyType string

const (
	loggerTraceDepth loggerKeyType = "loggerTraceDepth"
	separator        string        = " "
)

var localTraceInfo sync.Map

type traceCtxInfo struct {
	level      int
	childCount int
	id         string
}

func (i *traceCtxInfo) incInfo() string {
	i.childCount++
	if i.childCount > 1 {
		return fmt.Sprintf("(%d.%d)", i.level, i.childCount-1)
	}
	return fmt.Sprintf("(%d)", i.level)
}

func withTraceInfo(parent context.Context) (context.Context, *traceCtxInfo) {
	info := fromContext(parent)

	newInfo := &traceCtxInfo{
		level:      1,
		childCount: 0,
		id:         uuid.New().String(),
	}
	ctx := parent
	if info != nil {
		newInfo.level = info.level + 1
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "tracing-parent", newInfo.id)
	// Update
	return context.WithValue(ctx, loggerTraceDepth, newInfo), newInfo
}

func fromContext(ctx context.Context) *traceCtxInfo {
	if rv, ok := ctx.Value(loggerTraceDepth).(*traceCtxInfo); ok {
		return rv
	}

	// Check metdata incoming for parent span and return it
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		value := md.Get("tracing-parent")
		if len(value) > 0 {
			if rv, ok := localTraceInfo.Load(value[len(value)-1]); ok {
				return rv.(*traceCtxInfo)
			}
		}
	}
	return nil
}

type logrusLogger struct {
	entry     *logrus.Entry
	span      opentracing.Span
	info      *traceCtxInfo
	operation string
}

func (s *logrusLogger) getSpan() string {
	spanStr := fmt.Sprintf("%v", s.span)
	if len(spanStr) > 0 && spanStr != "{}" && s.span != nil {
		return fmt.Sprintf(" span=%v", spanStr)
	}
	return ""
}

func (s *logrusLogger) getTraceInfo() string {
	if s.info != nil {
		return fmt.Sprintf("%v %v ", s.info.incInfo(), strings.Repeat(separator, s.info.level))
	}
	return ""
}

func (s *logrusLogger) Info(v ...interface{}) {
	s.log("[INFO]", v...)
}

func (s *logrusLogger) Infof(format string, v ...interface{}) {
	s.logf("[INFO]", format, v...)
}

func (s *logrusLogger) Warn(v ...interface{}) {
	s.log("[WARN]", v...)
}

func (s *logrusLogger) Warnf(format string, v ...interface{}) {
	s.logf("[WARN]", format, v...)
}

func (s *logrusLogger) Error(v ...interface{}) {
	s.log("[ERROR]", v...)
}

func (s *logrusLogger) Errorf(format string, v ...interface{}) {
	s.logf("[ERROR]", format, v...)
}

func (s *logrusLogger) Fatal(v ...interface{}) {
	s.log("[FATAL]", v...)
}

func (s *logrusLogger) Fatalf(format string, v ...interface{}) {
	s.logf("[FATAL]", format, v...)
}

func (s *logrusLogger) Debug(v ...interface{}) {
	s.log("[DEBUG]", v...)
}

func (s *logrusLogger) Debugf(format string, v ...interface{}) {
	s.logf("[DEBUG]", format, v...)
}

func (s *logrusLogger) Trace(v ...interface{}) {
	s.log("[TRACE]", v...)
}

func (s *logrusLogger) Tracef(format string, v ...interface{}) {
	s.logf("[TRACE]", format, v...)
}

func (s *logrusLogger) Object(k, v interface{}) {
	msg := ""
	cc, err := json.Marshal(v)
	if err == nil {
		msg = string(cc)
	} else {
		msg = fmt.Sprint(v)
	}
	s.Infof("%v=%s", k, msg)
}

func (s *logrusLogger) log(level string, v ...interface{}) {
	s.logf(level, "%s", fmt.Sprint(v...))
}

func (s *logrusLogger) logf(level, format string, v ...interface{}) {
	s.entry.Tracef("%s %s%s%v", level, s.getTraceInfo(), fmt.Sprintf(format, v...), s.getSpan())
}

func (s *logrusLogger) WithField(key, value interface{}) logger.Logger {
	newLog := s.entry.WithField(key.(string), value)
	log := &logrusLogger{
		entry:     newLog,
		span:      s.span,
		operation: s.operation,
		info:      s.info,
	}
	return log
}

// New - creates a logruslogger and returns context with it
func New(ctx context.Context) (context.Context, logger.Logger) {
	logrus.SetLevel(logrus.TraceLevel)
	entry := logrus.WithTime(time.Now()).WithFields(logger.Fields(ctx))
	newLog := &logrusLogger{
		entry: entry,
	}
	return logger.WithLog(ctx, newLog), newLog
}

// FromSpan - creates a new logruslogger from context, operation and span
// and returns context with it, logger, and a function to defer
func FromSpan(ctx context.Context, span opentracing.Span, operation string) (context.Context, logger.Logger, func()) {
	logrus.SetLevel(logrus.TraceLevel)
	entry := logrus.WithTime(time.Now()).WithFields(logger.Fields(ctx))

	var info *traceCtxInfo
	ctx, info = withTraceInfo(ctx)
	localTraceInfo.Store(info.id, info)

	newLog := &logrusLogger{
		entry:     entry,
		span:      span,
		operation: operation,
		info:      info,
	}

	newLog.printStart()
	return logger.WithLog(ctx, newLog), newLog, func() { localTraceInfo.Delete(info.id) }
}

func (s *logrusLogger) printStart() {
	prefix := strings.Repeat(separator, s.info.level)
	s.entry.Tracef("%s %v%s⎆ %v()%v", "[INFO]", s.info.incInfo(), prefix, s.operation, s.getSpan())
}
