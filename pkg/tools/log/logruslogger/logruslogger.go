// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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
	"google.golang.org/grpc/metadata"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/spanlogger"
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
	span      spanlogger.Span
	info      *traceCtxInfo
	operation string
}

func (s *logrusLogger) getSpan() string {
	if s.span != nil {
		spanStr := s.span.ToString()
		if len(spanStr) > 0 && spanStr != "{}" {
			return fmt.Sprintf(" span=%v", spanStr)
		}
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
	s.entry.Info(s.format("%s", v...))
}

func (s *logrusLogger) Infof(format string, v ...interface{}) {
	s.entry.Info(s.format(format, v...))
}

func (s *logrusLogger) Warn(v ...interface{}) {
	s.entry.Warn(s.format("%s", v...))
}

func (s *logrusLogger) Warnf(format string, v ...interface{}) {
	s.entry.Warn(s.format(format, v...))
}

func (s *logrusLogger) Error(v ...interface{}) {
	s.entry.Error(s.format("%s", v...))
}

func (s *logrusLogger) Errorf(format string, v ...interface{}) {
	s.entry.Error(s.format(format, v...))
}

func (s *logrusLogger) Fatal(v ...interface{}) {
	s.entry.Fatal(s.format("%s", v...))
}

func (s *logrusLogger) Fatalf(format string, v ...interface{}) {
	s.entry.Fatal(s.format(format, v...))
}

func (s *logrusLogger) Debug(v ...interface{}) {
	s.entry.Debug(s.format("%s", v...))
}

func (s *logrusLogger) Debugf(format string, v ...interface{}) {
	s.entry.Debug(s.format(format, v...))
}

func (s *logrusLogger) Trace(v ...interface{}) {
	s.entry.Trace(s.format("%s", v...))
}

func (s *logrusLogger) Tracef(format string, v ...interface{}) {
	s.entry.Trace(s.format(format, v...))
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
		entry:     newLog,
		span:      s.span,
		operation: s.operation,
		info:      s.info,
	}
	return logger
}

// New - creates a logruslogger and returns it
func New(ctx context.Context, fields ...logrus.Fields) log.Logger {
	entry := logrus.NewEntry(logrus.StandardLogger())
	for _, f := range fields {
		entry = entry.WithFields(f)
	}
	entry.Logger.SetFormatter(newFormatter())

	newLog := &logrusLogger{
		entry: entry,
	}
	return newLog
}

// FromSpan - creates a new logruslogger from context, operation and span
// and returns context with it, logger, and a function to defer
func FromSpan(ctx context.Context, span spanlogger.Span, operation string, fields map[string]interface{}) (context.Context, log.Logger, func()) {
	entry := logrus.WithFields(fields)
	entry.Logger.SetFormatter(newFormatter())

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
	return ctx, newLog, func() { localTraceInfo.Delete(info.id) }
}

func (s *logrusLogger) printStart() {
	prefix := strings.Repeat(separator, s.info.level)
	s.entry.Tracef("%v%s⎆ %v()%v", s.info.incInfo(), prefix, s.operation, s.getSpan())
}

func (s *logrusLogger) format(format string, v ...interface{}) string {
	return fmt.Sprintf("%s%s%s", s.getTraceInfo(), fmt.Sprintf(format, v...), s.getSpan())
}
