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

type loggerLevel int
type entriesType map[interface{}]interface{}
type contextKeyType string
type traceLoggerKeyType string

const (
	INFO loggerLevel = iota
	WARN
	ERROR
	FATAL
	CTXKEY_LOGGER          contextKeyType     = "logger"
	CTXKEY_LOGRUS_ENTRY    contextKeyType     = "logrusEntry"
	SPANLOGGER_OP_UNTITLED string             = "Untitled operation"
	traceLoggerTraceDepth  traceLoggerKeyType = "traceLoggerTraceDepth"
	maxStringLength        int                = 1000
	dotCount               int                = 3
	separator              string             = " "
	startSeparator         string             = " "
)

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

//Returns logger from context
func Log(ctx context.Context) Logger {
	if ctx != nil {
		if value := ctx.Value(CTXKEY_LOGGER); value != nil {
			return value.(Logger)
		}
	}
	return nil
}
