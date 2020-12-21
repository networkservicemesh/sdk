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

// Package logger provides an unified interface Logger for logging
// And also contains logrusLogger, spanLogger and traceLogger which implement it
package logger

import (
	"context"
)

type contextKeyType string

const (
	ctxKeyLogger   contextKeyType = "ctxKeyLogger"
	ctxKeyLogEntry contextKeyType = "ctxKeyLogEntry"
)

// Logger - unified interface for logging
type Logger interface {
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Warn(v ...interface{})
	Warnf(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})

	WithField(key, value interface{}) Logger
}

// Log - returns logger from context
func Log(ctx context.Context) Logger {
	if ctx != nil {
		if value := ctx.Value(ctxKeyLogger); value != nil {
			return value.(Logger)
		}
	}
	panic("Could not return Logger from Context")
}

// WithLog - creates new context with a Logger in it
func WithLog(ctx context.Context, log ...Logger) context.Context {
	/*
		if ctx != nil {
			if value := ctx.Value(ctxKeyLogger); value != nil {
				if group, ok := value.(*groupLogger); ok {
					loggers := make([]Logger, len(group.loggers) + len(log))
					group.loggers
					for i := range log{
						group
					}
				}
				return value.(Logger)
			}
		}
		panic("context is nil")
	*/
	return context.WithValue(ctx, ctxKeyLogger, log)
}

// WithFields - adds fields to the context
func WithFields(ctx context.Context, fields map[string]string) context.Context {
	return context.WithValue(ctx, ctxKeyLogEntry, fields)
}
