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

// Package trace provides a wrapper for tracing around a registry.{Registry,Discovery}{Server,Client}
package trace

import (
	"context"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"
	"github.com/networkservicemesh/sdk/pkg/tools/spanlogger"
	"github.com/networkservicemesh/sdk/pkg/tools/tracelogger"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func newLogger(ctx context.Context, operation string) (logger.Logger, context.Context, func()) {
	span, ctx, s, done := spanlogger.New(ctx, operation)
	trace, ctx := tracelogger.New(ctx, operation, s)
	ctx = logger.WithLog(ctx, span, trace)
	return logger.Log(ctx), ctx, done
}
