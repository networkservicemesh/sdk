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
)

type groupLogger struct {
	loggers []Logger
}

// newGroup - creates a new GroupLogger from slice of Logger
func newGroup(s ...Logger) Logger {
	loggers := make([]Logger, len(s))
	for i, t := range s {
		if loggers[i] = t; loggers[i] == nil {
			return nil
		}
	}
	return &groupLogger{loggers: loggers}
}

// NewLogger - creates a groupLogger consisting of a spanLogger and a traceLogger
func NewLogger(ctx context.Context, operation string) (Logger, context.Context, func(Logger)) {
	span, ctx := NewSpan(ctx, operation)
	trace, ctx := NewTrace(ctx, operation, span.(*spanLogger).Span())
	group := newGroup(span, trace)
	ctx = context.WithValue(ctx, ctxKeyLogger, group)
	return group, ctx, finish
}

// Close - finishes spanLogger's span
func finish(log Logger) {
	log.(*groupLogger).loggers[0].(*spanLogger).Finish()
}

func (j *groupLogger) Info(v ...interface{}) {
	for _, l := range j.loggers {
		l.Info(v...)
	}
}

func (j *groupLogger) Infof(format string, v ...interface{}) {
	for _, l := range j.loggers {
		l.Infof(format, v...)
	}
}

func (j *groupLogger) Warn(v ...interface{}) {
	for _, l := range j.loggers {
		l.Warn(v...)
	}
}

func (j *groupLogger) Warnf(format string, v ...interface{}) {
	for _, l := range j.loggers {
		l.Warnf(format, v...)
	}
}

func (j *groupLogger) Error(v ...interface{}) {
	for _, l := range j.loggers {
		l.Error(v...)
	}
}

func (j *groupLogger) Errorf(format string, v ...interface{}) {
	for _, l := range j.loggers {
		l.Errorf(format, v...)
	}
}

func (j *groupLogger) Fatal(v ...interface{}) {
	for _, l := range j.loggers {
		l.Fatal(v...)
	}
}

func (j *groupLogger) Fatalf(format string, v ...interface{}) {
	for _, l := range j.loggers {
		l.Fatalf(format, v...)
	}
}

func (j *groupLogger) WithField(key, value interface{}) Logger {
	loggers := make([]Logger, len(j.loggers))
	for i, l := range j.loggers {
		loggers[i] = l.WithField(key, value)
	}
	return &groupLogger{loggers: loggers}
}
