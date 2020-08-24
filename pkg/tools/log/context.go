// Copyright (c) 2020 Cisco and/or its affiliates.
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

// Package log provides functions for having a *logrus.Entry per Context
// This allows the for context associative logging
package log

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type contextKeyType string

const (
	logrusEntry contextKeyType = "LogrusEntry"
)

// WithFields - return new context with fields added to Context's logrus.Entry
func WithFields(parent context.Context, fields logrus.Fields) context.Context {
	entry := Entry(parent)
	entry = entry.WithFields(fields)
	ctx := context.WithValue(parent, logrusEntry, entry)
	entry.Context = ctx
	return ctx
}

// WithField - return new context with {key:value} added to Context's logrus.Entry
func WithField(parent context.Context, key string, value interface{}) context.Context {
	return WithFields(parent, logrus.Fields{
		key: value,
	})
}

// Entry - returns *logrus.Entry for context.  Note: each context has its *own* Entry with Entry.Context set
//         to that context (so context values can be used in logrus.Hooks)
func Entry(ctx context.Context) *logrus.Entry {
	if ctx == nil {
		panic("cannot create context from nil parent")
	}
	if entryValue := ctx.Value(logrusEntry); entryValue != nil {
		if entry := entryValue.(*logrus.Entry); entry != nil {
			if entry.Context == ctx {
				return entry
			}
			return entry.WithContext(ctx)
		}
	}
	return logrus.WithTime(time.Now())
}
