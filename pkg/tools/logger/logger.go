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

// Package logger provides an unified interface Logger for logging
package logger

import (
	"context"
)

type contextKeyType string

const (
	logKey       contextKeyType = "Logger"
	logFieldsKey contextKeyType = "LoggerFields"
)

var (
	isTracingEnabled = false
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
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	Trace(v ...interface{})
	Tracef(format string, v ...interface{})

	Object(k, v interface{})

	WithField(key, value interface{}) Logger
}

// Log - returns logger from context
func Log(ctx context.Context) Logger {
	rv, ok := ctx.Value(logKey).(Logger)
	if ok {
		return rv
	}
	panic("Could not return Logger from Context")
}

// WithLog - creates new context with `log` in it
func WithLog(ctx context.Context, log ...Logger) context.Context {
	return context.WithValue(ctx, logKey, newGroup(log...))
}

// Fields - returns fields from context
func Fields(ctx context.Context) map[string]interface{} {
	if rv := ctx.Value(logFieldsKey); rv != nil {
		return rv.(map[string]interface{})
	}
	return nil
}

// WithFields - adds/replaces fields to/in context
func WithFields(ctx context.Context, fields map[string]interface{}) context.Context {
	return context.WithValue(ctx, logFieldsKey, fields)
}

// IsTracingEnabled - checks if it is allowed to use traces
func IsTracingEnabled() bool {
	return isTracingEnabled
}

// EnableTracing - enable/disable traces
func EnableTracing(enable bool) {
	isTracingEnabled = enable
}
