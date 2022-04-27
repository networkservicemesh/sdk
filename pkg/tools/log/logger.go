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

// Package log provides an unified interface Logger for logging
package log

import (
	"context"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

type contextKeyType string

const (
	logKey contextKeyType = "Logger"
)

var (
	isTracingEnabled int32 = 0
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

// FromContext - returns logger from context
func FromContext(ctx context.Context) Logger {
	rv, ok := ctx.Value(logKey).(Logger)
	if ok {
		return rv
	}
	return Default()
}

// Join - concatenates new logger with existing loggers
func Join(ctx context.Context, log Logger) context.Context {
	rv, ok := ctx.Value(logKey).(Logger)
	if ok {
		return WithLog(ctx, rv, log)
	}
	return WithLog(ctx, log)
}

// WithLog - creates new context with `log` inside
func WithLog(ctx context.Context, log ...Logger) context.Context {
	return context.WithValue(ctx, logKey, Combine(log...))
}

// IsTracingEnabled - checks if it is allowed to use traces
func IsTracingEnabled() bool {
	// TODO: Rework this within https://github.com/networkservicemesh/sdk/issues/1272
	return atomic.LoadInt32(&isTracingEnabled) != 0 && logrus.GetLevel() == logrus.TraceLevel
}

// EnableTracing - enable/disable traces
func EnableTracing(enable bool) {
	if enable {
		atomic.StoreInt32(&isTracingEnabled, 1)
		return
	}

	atomic.StoreInt32(&isTracingEnabled, 0)
}
