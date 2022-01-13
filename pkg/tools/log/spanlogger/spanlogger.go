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

// Package spanlogger provides a set of utilities to assist in working with spans
package spanlogger

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/opentelemetry"
)

// spanlogger - provides a way to log via opentelemetry spans
type spanLogger struct {
	span Span
	lock sync.RWMutex
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
	s.log("error", v...)
}

func (s *spanLogger) Errorf(format string, v ...interface{}) {
	s.logf("error", format, v...)
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
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.span != nil {
		if v != nil {
			msg := ""
			cc, err := json.Marshal(v)
			if err == nil {
				msg = string(cc)
			} else {
				msg = fmt.Sprint(v)
			}
			s.span.LogObject(k, msg)
		}
	}
}

func (s *spanLogger) WithField(key, value interface{}) log.Logger {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.span != nil {
		newlog := &spanLogger{
			span: s.span.WithField(key, value),
		}
		return newlog
	}
	return s
}

func (s *spanLogger) log(level string, v ...interface{}) {
	s.logf(level, "%s", fmt.Sprint(v...))
}

func (s *spanLogger) logf(level, format string, v ...interface{}) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.span != nil {
		if v != nil {
			s.span.Log(level, format, v...)
		}
	}
}

// FromContext - creates a new spanLogger from context and operation
func FromContext(ctx context.Context, operation string, fields map[string]interface{}) (context.Context, log.Logger, Span, func()) {
	var span Span
	if opentelemetry.IsEnabled() {
		ctx, span = newOTELSpan(ctx, operation, fields)
	}
	newLog := &spanLogger{
		span: span,
	}
	return ctx, newLog, span, func() { newLog.finish() }
}

// finish - closes spanLogger
func (s *spanLogger) finish() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.span != nil {
		s.span.Finish()
		s.span = nil
	}
}
