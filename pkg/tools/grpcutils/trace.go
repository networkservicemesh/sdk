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

package grpcutils

import (
	"context"
	"strconv"

	"google.golang.org/grpc/metadata"
)

// TraceState is a type that defines the state of the traces stored in grpc metadata
type TraceState int

const (
	// TraceUndefined - no state is defined
	TraceUndefined TraceState = iota

	// TraceOn - tracing is enabled
	TraceOn

	// TraceOff - tracing is disabled
	TraceOff
)

const (
	grpcTraceKey string = "GrpcTracing"
)

// TraceFromContext - checks if incoming metadata allows traces
func TraceFromContext(ctx context.Context) TraceState {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get(grpcTraceKey)
		if len(values) > 0 {
			val, err := strconv.Atoi(values[len(values)-1])
			if err != nil {
				return TraceUndefined
			}
			return TraceState(val)
		}
	}
	return TraceUndefined
}

// WithTrace - enable/disable traces for outgoing context
func WithTrace(ctx context.Context, state TraceState) context.Context {
	return metadata.AppendToOutgoingContext(ctx, grpcTraceKey, strconv.Itoa(int(state)))
}

// PassTraceToOutgoing - passes trace state from incoming to outgoing context
func PassTraceToOutgoing(ctx context.Context) context.Context {
	if !hasOutgoingTrace(ctx) {
		return WithTrace(ctx, TraceFromContext(ctx))
	}
	return ctx
}

// hasOutgoingTrace - checks if outgoing context already has trace state
func hasOutgoingTrace(ctx context.Context) bool {
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		values := md.Get(grpcTraceKey)
		if len(values) > 0 {
			return true
		}
	}
	return false
}
