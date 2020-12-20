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

// NewLogrus - returns a new logrusLogger based on logrus.Entry from context
func NewLogrus(ctx context.Context) (Logger, context.Context) {
	var fields map[string]string = nil
	if value, ok := ctx.Value(ctxKeyLogEntry).(map[string]string); ok {
		fields = value
	}
	entry := logrus.WithTime(time.Now()).WithContext(ctx)
	for k, v := range fields {
		entry = entry.WithField(k, v)
	}
	log := &logrusLogger{entry: entry}
	ctx = context.WithValue(ctx, ctxKeyLogger, log)
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

func (r *logrusLogger) WithField(key, value interface{}) Logger {
	entry := r.entry
	entry = entry.WithFields(logrus.Fields{key.(string): value})
	log := &logrusLogger{entry: entry}
	return log
}
