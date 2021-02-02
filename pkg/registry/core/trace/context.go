// Copyright (c) 2020 Cisco Systems, Inc.
//
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

package trace

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/logger"
	"github.com/networkservicemesh/sdk/pkg/tools/logger/tracelogger"
)

// withLog - provides corresponding logger in context
func withLog(parent context.Context, operation string) (c context.Context, f func()) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}

	// Update outgoing grpc context
	parent = grpcutils.PassTraceToOutgoing(parent)

	if grpcTraceState := grpcutils.TraceFromContext(parent); (grpcTraceState == grpcutils.TraceOn) ||
		(grpcTraceState == grpcutils.TraceUndefined && logger.IsTracingEnabled()) {
		return tracelogger.WithLog(parent, operation)
	}
	return logger.WithLog(parent), func() {}
}
