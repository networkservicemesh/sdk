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
package logruslogger

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"
)

const (
	// CtxKeyLogEntry - context key for entries map
	CtxKeyLogEntry logger.ContextKeyType = "ctxKeyLogEntry"
)

// New - returns a new logrusLogger based on logrus.Entry from context
func New(ctx context.Context) (logger.Logger, context.Context) {
	var fields map[string]string = nil
	if value, ok := ctx.Value(CtxKeyLogEntry).(map[string]string); ok {
		fields = value
	}
	entry := logrus.WithTime(time.Now()).WithContext(ctx)
	for k, v := range fields {
		entry = entry.WithField(k, v)
	}
	log := &logrusLogger{entry: entry}
	ctx = logger.WithLog(ctx, log)
	return log, ctx
}

type logrusLogger struct {
	entry *logrus.Entry
}

func (r *logrusLogger) Info(v ...interface{}) {
	r.entry.Info(v...)
}

func (r *logrusLogger) Infof(format string, v ...interface{}) {
	r.entry.Infof(format, v...)
}

func (r *logrusLogger) Warn(v ...interface{}) {
	r.entry.Warn(v...)
}

func (r *logrusLogger) Warnf(format string, v ...interface{}) {
	r.entry.Warnf(format, v...)
}

func (r *logrusLogger) Error(v ...interface{}) {
	r.entry.Error(v...)
}

func (r *logrusLogger) Errorf(format string, v ...interface{}) {
	r.entry.Errorf(format, v...)
}

func (r *logrusLogger) Fatal(v ...interface{}) {
	r.entry.Error(v...)
}

func (r *logrusLogger) Fatalf(format string, v ...interface{}) {
	r.entry.Errorf(format, v...)
}

func (r *logrusLogger) WithField(key, value interface{}) logger.Logger {
	entry := r.entry
	entry = entry.WithFields(logrus.Fields{key.(string): value})
	log := &logrusLogger{entry: entry}
	return log
}

// WithFields - adds fields to the context
func WithFields(ctx context.Context, fields map[string]string) context.Context {
	return context.WithValue(ctx, CtxKeyLogEntry, fields)
}
