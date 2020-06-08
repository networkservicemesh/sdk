// Copyright (c) 2020 Cisco Systems, Inc.
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

// Package trace provides a wrapper for tracing around a networkservice.NetworkServiceClient
package trace

import (
	"context"

	"github.com/sirupsen/logrus"
)

type contextKeyType string

const (
	logKey contextKeyType = "Log"
)

// withLog -
//   Provides a FieldLogger in context
func withLog(parent context.Context, log logrus.FieldLogger) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, logKey, log)
}

// Log - return FieldLogger from context
func Log(ctx context.Context) logrus.FieldLogger {
	rv, ok := ctx.Value(logKey).(logrus.FieldLogger)
	if !ok {
		logger := &logrus.Logger{
			Out:          logrus.StandardLogger().Out,
			Formatter:    logrus.StandardLogger().Formatter,
			Hooks:        make(logrus.LevelHooks),
			Level:        logrus.StandardLogger().Level,
			ExitFunc:     logrus.StandardLogger().ExitFunc,
			ReportCaller: logrus.StandardLogger().ReportCaller,
		}
		for k, v := range logrus.StandardLogger().Hooks {
			logger.Hooks[k] = v
		}
		rv = logger
	}
	return rv
}
