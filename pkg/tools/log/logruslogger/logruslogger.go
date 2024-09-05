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

// Package logruslogger provides wrapper for logrus logger
// which is consistent with Logger interface
package logruslogger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.uber.org/atomic"
	"google.golang.org/grpc/metadata"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/spanlogger"
)

type logrusLogger struct {
	entry *logrus.Entry
}

func (s *logrusLogger) Info(v ...interface{}) {
	s.entry.Info(v...)
}

func (s *logrusLogger) Infof(format string, v ...interface{}) {
	s.entry.Infof(format, v...)
}

func (s *logrusLogger) Warn(v ...interface{}) {
	s.entry.Warn(v...)
}

func (s *logrusLogger) Warnf(format string, v ...interface{}) {
	s.entry.Warnf(format, v...)
}

func (s *logrusLogger) Error(v ...interface{}) {
	s.entry.Error(v...)
}

func (s *logrusLogger) Errorf(format string, v ...interface{}) {
	s.entry.Errorf(format, v...)
}

func (s *logrusLogger) Fatal(v ...interface{}) {
	s.entry.Fatal(v...)
}

func (s *logrusLogger) Fatalf(format string, v ...interface{}) {
	s.entry.Fatalf(format, v...)
}

func (s *logrusLogger) Debug(v ...interface{}) {
	s.entry.Debug(v...)
}

func (s *logrusLogger) Debugf(format string, v ...interface{}) {
	s.entry.Debugf(format, v...)
}

func (s *logrusLogger) Trace(v ...interface{}) {
	s.entry.Trace(v...)
}

func (s *logrusLogger) Tracef(format string, v ...interface{}) {
	s.entry.Tracef(format, v...)
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

func (s *logrusLogger) WithField(key, value interface{}) log.Logger {
	newLog := s.entry.WithField(key.(string), value)
	logger := &logrusLogger{
		entry: newLog,
	}
	return logger
}

// New - create a logruslogger.
func New(ctx context.Context, fields ...logrus.Fields) log.Logger {
	entry := logrus.NewEntry(logrus.StandardLogger())
	for _, f := range fields {
		entry = entry.WithFields(f)
	}
	entry.Logger.SetFormatter(newFormatter())

	return &logrusLogger{
		entry: entry,
	}
}

// ----------------------------------------------------------------------

type traceLogger struct {
	logger    log.Logger
	span      spanlogger.Span
	info      *traceCtxInfo
	operation string
}

type loggerKeyType string

const (
	loggerTraceDepth loggerKeyType = "loggerTraceDepth"
	separator        string        = " "
)

var localTraceInfo sync.Map

type traceCtxInfo struct {
	level      int
	childCount atomic.Int32
	id         string
}

func (s *traceLogger) Info(v ...interface{}) {
	s.logger.Info(s.format("%s", v...))
}

func (s *traceLogger) Infof(format string, v ...interface{}) {
	s.logger.Infof(s.format(format, v...))
}

func (s *traceLogger) Warn(v ...interface{}) {
	s.logger.Warn(s.format("%s", v...))
}

func (s *traceLogger) Warnf(format string, v ...interface{}) {
	s.logger.Warnf(s.format(format, v...))
}

func (s *traceLogger) Error(v ...interface{}) {
	s.logger.Error(s.format("%s", v...))
}

func (s *traceLogger) Errorf(format string, v ...interface{}) {
	s.logger.Error(s.format(format, v...))
}

func (s *traceLogger) Fatal(v ...interface{}) {
	s.logger.Fatal(s.format("%s", v...))
}

func (s *traceLogger) Fatalf(format string, v ...interface{}) {
	s.logger.Fatal(s.format(format, v...))
}

func (s *traceLogger) Debug(v ...interface{}) {
	s.logger.Debug(s.format("%s", v...))
}

func (s *traceLogger) Debugf(format string, v ...interface{}) {
	s.logger.Debug(s.format(format, v...))
}

func (s *traceLogger) Trace(v ...interface{}) {
	s.logger.Trace(s.format("%s", v...))
}

func (s *traceLogger) Tracef(format string, v ...interface{}) {
	s.logger.Trace(s.format(format, v...))
}

func (s *traceLogger) Object(k, v interface{}) {
	msg := ""
	cc, err := json.Marshal(v)
	if err == nil {
		msg = string(cc)
	} else {
		msg = fmt.Sprint(v)
	}
	s.Infof("%v=%s", k, msg)
}

func (s *traceLogger) WithField(key, value interface{}) log.Logger {
	newLog := s.logger.WithField(key.(string), value)
	logger := &traceLogger{
		logger:    newLog,
		span:      s.span,
		operation: s.operation,
		info:      s.info,
	}
	return logger
}

func (i *traceCtxInfo) incInfo() string {
	newValue := i.childCount.Inc()
	if newValue > 1 {
		return fmt.Sprintf("(%d.%d)", i.level, newValue-1)
	}
	return fmt.Sprintf("(%d)", i.level)
}

func withTraceInfo(parent context.Context) (context.Context, *traceCtxInfo) {
	newInfo := &traceCtxInfo{
		level:      1,
		childCount: atomic.Int32{},
		id:         uuid.New().String(),
	}

	if info := fromContext(parent); info != nil {
		newInfo.level = info.level + 1
	}
	ctx := metadata.AppendToOutgoingContext(parent, "tracing-parent", newInfo.id)
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

func (s *traceLogger) getSpan() string {
	if s.span != nil {
		spanStr := s.span.ToString()
		if spanStr != "" && spanStr != "{}" {
			return fmt.Sprintf(" span=%v", spanStr)
		}
	}
	return ""
}

func (s *traceLogger) getTraceInfo() string {
	if s.info != nil {
		return fmt.Sprintf("%v %v ", s.info.incInfo(), strings.Repeat(separator, s.info.level))
	}
	return ""
}

// FromSpan - creates a new logruslogger from context, operation and span
// and returns context with it, logger, and a function to defer.
func FromSpan(
	ctx context.Context, span spanlogger.Span, operation string, fields []*log.Field,
) (context.Context, log.Logger, func()) {
	var info *traceCtxInfo
	deleteFunc := func() {}
	if log.IsTracingEnabled() && logrus.GetLevel() == logrus.TraceLevel {
		ctx, info = withTraceInfo(ctx)
		localTraceInfo.Store(info.id, info)
		deleteFunc = func() { localTraceInfo.Delete(info.id) }
	}

	logger := log.L()
	if log.IsDefault(logger) {
		fieldsMap := make(map[string]interface{}, len(fields))
		for _, field := range fields {
			fieldsMap[field.Key()] = field.Val()
		}
		entry := logrus.WithFields(fieldsMap)
		entry.Logger.SetFormatter(newFormatter())
		logger = &logrusLogger{
			entry: entry,
		}
	} else {
		for _, field := range fields {
			logger = logger.WithField(field.Key(), field.Val())
		}
	}

	newLog := &traceLogger{
		logger:    logger,
		span:      span,
		operation: operation,
		info:      info,
	}

	newLog.printStart()
	return ctx, newLog, deleteFunc
}

func (s *traceLogger) printStart() {
	if s.info != nil {
		prefix := strings.Repeat(separator, s.info.level)
		s.logger.Tracef("%v%s⎆ %v()%v", s.info.incInfo(), prefix, s.operation, s.getSpan())
	}
}

func (s *traceLogger) format(format string, v ...interface{}) string {
	return fmt.Sprintf("%s%s%s", s.getTraceInfo(), fmt.Sprintf(format, v...), s.getSpan())
}
