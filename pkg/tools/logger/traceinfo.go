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
	"fmt"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

var localTraceInfo sync.Map

type traceLoggerKeyType string

const (
	traceLoggerTraceDepth traceLoggerKeyType = "traceLoggerTraceDepth"
)

type traceCtxInfo struct {
	level      int
	childCount int
	id         string
}

func (i *traceCtxInfo) incInfo() string {
	i.childCount++
	if i.childCount > 1 {
		return fmt.Sprintf("(%d.%d)", i.level, i.childCount-1)
	}
	return fmt.Sprintf("(%d)", i.level)
}

func withTraceInfo(parent context.Context) (context.Context, *traceCtxInfo) {
	info := fromContext(parent)

	newInfo := &traceCtxInfo{
		level:      1,
		childCount: 0,
		id:         uuid.New().String(),
	}
	ctx := parent
	if info != nil {
		newInfo.level = info.level + 1
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "tracelogger-parent", newInfo.id)
	// Update
	return context.WithValue(ctx, traceLoggerTraceDepth, newInfo), newInfo
}

func fromContext(ctx context.Context) *traceCtxInfo {
	if rv, ok := ctx.Value(traceLoggerTraceDepth).(*traceCtxInfo); ok {
		return rv
	}

	// Check metdata incoming for parent span and return it
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		value := md.Get("tracelogger-parent")
		if len(value) > 0 {
			if rv, ok := localTraceInfo.Load(value[len(value)-1]); ok {
				return rv.(*traceCtxInfo)
			}
		}
	}

	return nil
}
