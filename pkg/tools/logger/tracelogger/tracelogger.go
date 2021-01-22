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

// Package tracelogger allows to create a composite Logger
// for traces
package tracelogger

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"
	"github.com/networkservicemesh/sdk/pkg/tools/logger/logruslogger"
)

// WithLog - creates a composite Logger which consists of logruslogger and spanlogger
// and returns context with it and a function to defer
func WithLog(ctx context.Context, operation string) (c context.Context, f func()) {
	ctx, sLogger, span, sFinish := newSpanLogger(ctx, operation)
	ctx, lLogger, lFinish := logruslogger.FromSpan(ctx, span, operation)
	return logger.WithLog(ctx, sLogger, lLogger), func() {
		sFinish()
		lFinish()
	}
}
