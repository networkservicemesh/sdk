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

// Package defaultlogger allows to create a composite Logger
// which consists of logruslogger and spanlogger
package defaultlogger

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"
	"github.com/networkservicemesh/sdk/pkg/tools/logruslogger"
	"github.com/networkservicemesh/sdk/pkg/tools/spanlogger"
)

// New - creates a composite Logger which consists of logruslogger and spanlogger
func New(ctx context.Context, operation string) (logger.Logger, context.Context, func()) {
	span, ctx, s, sdone := spanlogger.New(ctx, operation)
	trace, ctx, tdone := logruslogger.FromSpan(ctx, operation, s)
	ctx = logger.WithLog(ctx, span, trace)
	return logger.Log(ctx), ctx, func() {
		sdone()
		tdone()
	}
}
