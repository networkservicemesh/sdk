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

// Package childspanlogger contains span logger for specific case of inifinite Find request
package childspanlogger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	opentracinglog "github.com/opentracing/opentracing-go/log"

	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	maxStringLength int = 1000
	dotCount        int = 3
)

var loggers []*childSpanLogger
var ticker = time.NewTicker(time.Minute)
var once sync.Once
var lock sync.Mutex

// childSpanlogger - provides a way to log via opentracing spans
type childSpanLogger struct {
	span      opentracing.Span
	childSpan opentracing.Span
	entries   map[interface{}]interface{}
	lock      sync.RWMutex
	operation string
}

func (s *childSpanLogger) Info(v ...interface{}) {
	s.log("info", v...)
}

func (s *childSpanLogger) Infof(format string, v ...interface{}) {
	s.logf("info", format, v...)
}

func (s *childSpanLogger) Warn(v ...interface{}) {
	s.log("warn", v...)
}

func (s *childSpanLogger) Warnf(format string, v ...interface{}) {
	s.logf("warn", format, v...)
}

func (s *childSpanLogger) Error(v ...interface{}) {
	s.log("error", v...)
}

func (s *childSpanLogger) Errorf(format string, v ...interface{}) {
	s.logf("error", format, v...)
}

func (s *childSpanLogger) Fatal(v ...interface{}) {
	s.log("fatal", v...)
}

func (s *childSpanLogger) Fatalf(format string, v ...interface{}) {
	s.logf("fatal", format, v...)
}

func (s *childSpanLogger) Debug(v ...interface{}) {
	s.log("debug", v...)
}

func (s *childSpanLogger) Debugf(format string, v ...interface{}) {
	s.logf("debug", format, v...)
}

func (s *childSpanLogger) Trace(v ...interface{}) {
	s.log("trace", v...)
}

func (s *childSpanLogger) Tracef(format string, v ...interface{}) {
	s.logf("trace", format, v...)
}

func (s *childSpanLogger) Object(k, v interface{}) {
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

			s.span.LogFields(opentracinglog.Object(k.(string), limitString(msg)))
			for k, v := range s.entries {
				s.span.LogKV(k, v)
			}
		}
	}
}

func (s *childSpanLogger) WithField(key, value interface{}) log.Logger {
	s.lock.RLock()
	defer s.lock.RUnlock()

	data := make(map[interface{}]interface{}, len(s.entries)+1)
	for k, v := range s.entries {
		data[k] = v
	}
	data[key] = value
	newlog := &childSpanLogger{
		span:      s.span,
		entries:   data,
		childSpan: s.childSpan,
		operation: s.operation,
	}
	return newlog
}

func (s *childSpanLogger) log(level string, v ...interface{}) {
	s.logf(level, "%s", fmt.Sprint(v...))
}

func (s *childSpanLogger) logf(level, format string, v ...interface{}) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.childSpan != nil {
		if v != nil {
			s.childSpan.LogFields(opentracinglog.String("event", level), opentracinglog.String("message", fmt.Sprintf(format, v...)))
			for k, v := range s.entries {
				s.childSpan.LogKV(k, v)
			}
		}
	}
}

// FromContext - creates a new childSpanLogger from context and operation
func FromContext(ctx context.Context, operation string) (context.Context, log.Logger, opentracing.Span, func()) {
	once.Do(func() {
		go func() {
			for {
				select {
				case <-ticker.C:
					for _, l := range loggers {
						if l.span != nil {
							l.lock.Lock()
							l.childSpan.Finish()
							l.childSpan = l.span.Tracer().StartSpan(l.operation, opentracing.ChildOf(l.span.Context()))
							l.lock.Unlock()
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	})

	var span opentracing.Span
	if jaeger.IsOpentracingEnabled() {
		span, ctx = opentracing.StartSpanFromContext(ctx, operation)
	}

	newLog := &childSpanLogger{
		span:      span,
		entries:   make(map[interface{}]interface{}),
		childSpan: span.Tracer().StartSpan(operation, opentracing.ChildOf(span.Context())),
		operation: operation,
	}

	lock.Lock()
	loggers = append(loggers, newLog)
	lock.Unlock()

	return ctx, newLog, span, func() { newLog.finish() }
}

// finish - closes childSpanLogger
func (s *childSpanLogger) finish() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.span != nil {
		s.childSpan.Finish()
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
