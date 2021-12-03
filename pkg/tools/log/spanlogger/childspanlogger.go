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

package spanlogger

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opentracing/opentracing-go"
	opentracinglog "github.com/opentracing/opentracing-go/log"

	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/spanlogger/notifcator"
)

// childSpanlogger - provides a way to log via opentracing spans
type childSpanLogger struct {
	spanLogger
	childSpan opentracing.Span
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

	if s.childSpan != nil {
		if v != nil {
			msg := ""
			cc, err := json.Marshal(v)
			if err == nil {
				msg = string(cc)
			} else {
				msg = fmt.Sprint(v)
			}

			s.childSpan.LogFields(opentracinglog.Object(k.(string), limitString(msg)))
			for k, v := range s.entries {
				s.childSpan.LogKV(k, v)
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
		spanLogger: spanLogger{
			span:    s.spanLogger.span,
			entries: data,
		},
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

// FromContextNonBlockingSpanLogger - creates a new childSpanLogger from context and operation
func FromContextNonBlockingSpanLogger(ctx context.Context, operation string) (context.Context, log.Logger, opentracing.Span, func()) {
	var span opentracing.Span
	if jaeger.IsOpentracingEnabled() {
		span, ctx = opentracing.StartSpanFromContext(ctx, operation)
	}

	var child opentracing.Span
	if span != nil {
		child = span.Tracer().StartSpan(operation, opentracing.ChildOf(span.Context()))
	}

	newLog := &childSpanLogger{
		spanLogger: spanLogger{
			span:    span,
			entries: make(map[interface{}]interface{}),
		},
		childSpan: child,
		operation: operation,
	}
	ch := make(chan struct{})
	ctx = notifcator.WithNotificator(ctx)
	not := notifcator.FromContext(ctx)
	if not != nil {
		not.AddSubscribers(ch)
		go func() {
			for {
				select {
				case <-ch:
					newLog.finishChild()
				case <-not.Done():
					newLog.finish()
					return
				}
			}
		}()
	}

	return ctx, newLog, span, func() { newLog.finish() }
}

// finish - closes childSpanLogger
func (s *childSpanLogger) finish() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.span != nil && s.childSpan != nil {
		s.childSpan.Finish()
		s.childSpan = nil
		s.span.Finish()
		s.span = nil
	}
}

func (s *childSpanLogger) finishChild() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.span != nil && s.childSpan != nil {
		s.childSpan.Finish()
		s.childSpan = s.span.Tracer().StartSpan(s.operation, opentracing.ChildOf(s.span.Context()))
	}
}
