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
	"time"

	"github.com/sirupsen/logrus"
)

func newLogrus(ctx context.Context, entry *logrus.Entry) Logger {
	logger := &logrusLogger{}
	ctx = context.WithValue(ctx, CTXKEY_LOGGER, logger)
	logger.entry = entry.WithContext(ctx)
	return logger
}

func NewLogrus(ctx context.Context) Logger {
	if value, ok := ctx.Value(CTXKEY_LOGRUS_ENTRY).(*logrus.Entry); ok {
		return newLogrus(ctx, value)
	}
	return newLogrus(ctx, logrus.WithTime(time.Now()))
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

func (r *logrusLogger) WithField(key, value interface{}) Logger {
	entry := r.entry
	entry = entry.WithFields(logrus.Fields{key.(string): value})
	logger := &logrusLogger{entry: entry}
	ctx := context.WithValue(entry.Context, CTXKEY_LOGRUS_ENTRY, entry)
	ctx = context.WithValue(ctx, CTXKEY_LOGGER, logger)
	entry.Context = ctx
	return logger
}

//Returns context with logger and logrus entry in it
func (s *logrusLogger) Context() context.Context {
	return s.entry.Context
}
