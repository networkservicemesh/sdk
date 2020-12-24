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

// Package logruslogger provides wrapper for logrus logger
// which is consistent with Logger interface
// and sends messages containing tracing information
package logruslogger

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"
)

type logrusLogger struct {
	mutex     *sync.Mutex
	operation string
	span      opentracing.Span
	info      *traceCtxInfo
	entry     *logrus.Entry
}

func (s *logrusLogger) Info(v ...interface{}) {
	s.log(v)
}

func (s *logrusLogger) Infof(format string, v ...interface{}) {
	s.logf(format, v...)
}

func (s *logrusLogger) Warn(v ...interface{}) {
	s.log(v)
}

func (s *logrusLogger) Warnf(format string, v ...interface{}) {
	s.logf(format, v...)
}

func (s *logrusLogger) Error(v ...interface{}) {
	s.log(v)
}

func (s *logrusLogger) Errorf(format string, v ...interface{}) {
	s.logf(format, v...)
}

func (s *logrusLogger) Fatal(v ...interface{}) {
	s.log(v)
}

func (s *logrusLogger) Fatalf(format string, v ...interface{}) {
	s.logf(format, v...)
}

// FromSpan - returns a new logrusLogger from context and span with given operation name
func FromSpan(ctx context.Context, operation string, span opentracing.Span) (logger.Logger, context.Context, func()) {
	entry := logrus.WithTime(time.Now()).WithContext(ctx)
	if fields := logger.Fields(ctx); fields != nil {
		for k, v := range fields {
			entry = entry.WithField(k.(string), v)
		}
	}
	var info *traceCtxInfo
	ctx, info = withTraceInfo(ctx)
	localTraceInfo.Store(info.id, info)
	log := &logrusLogger{
		mutex:     &sync.Mutex{},
		span:      span,
		info:      info,
		operation: operation,
		entry:     entry,
	}
	log.printStart(operation)
	return log, ctx, func() { localTraceInfo.Delete(info.id) }
}

// New - returns a new logrusLogger from context
func New(ctx context.Context) (logger.Logger, context.Context, func()) {
	return FromSpan(ctx, "", nil)
}

func (s *logrusLogger) WithField(key, value interface{}) logger.Logger {
	entry := s.entry
	entry = entry.WithFields(logrus.Fields{key.(string): value})
	s.mutex.Lock()
	log := &logrusLogger{mutex: s.mutex, span: s.span, entry: entry, info: s.info}
	s.mutex.Unlock()
	return log
}

func (s *logrusLogger) log(v ...interface{}) {
	s.logf(format(v), v)
}

func (s *logrusLogger) logf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	s.mutex.Lock()
	incInfo := s.info.incInfo()
	s.entry.Tracef("%v %s %v%v", incInfo, strings.Repeat(" ", s.info.level), msg, s.getSpan())
	s.mutex.Unlock()
}

func (s *logrusLogger) getSpan() string {
	spanStr := fmt.Sprintf("%v", s.span)
	if len(spanStr) > 0 && spanStr != "{}" && s.span != nil {
		return fmt.Sprintf(" span=%v", spanStr)
	}
	return ""
}

func (s *logrusLogger) printStart(operation string) {
	if operation == "" {
		return
	}
	prefix := strings.Repeat(" ", s.info.level)
	s.mutex.Lock()
	incInfo := s.info.incInfo()
	s.mutex.Unlock()
	s.entry.Tracef("%v%s⎆ %v()%v", incInfo, prefix, operation, s.getSpan())
}

func format(v ...interface{}) string {
	return strings.Trim(strings.Repeat("%+v ", len(v)), " ")
}
