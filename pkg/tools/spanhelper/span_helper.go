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

// Package spanhelper provides a set of utilities to assist in working with opentracing spans
package spanhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"

	toolsLog "github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
)

type spanHelperKeyType string

const (
	spanHelperTraceDepth spanHelperKeyType = "spanHelperTraceDepth"
	spanHelperState      spanHelperKeyType = "spanHelperState"
	maxStringLength      int               = 1000
	dotCount             int               = 3
	separator            string            = " "
	startSeparator       string            = " "

	spanField      string = "span"
	operationField string = "operation"
)

var localTraceInfo sync.Map
var hookInit sync.Once

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

	ctx = metadata.AppendToOutgoingContext(ctx, "spanhelper-parent", newInfo.id)
	// Update
	return context.WithValue(ctx, spanHelperTraceDepth, newInfo), newInfo
}

func fromContext(ctx context.Context) *traceCtxInfo {
	if rv, ok := ctx.Value(spanHelperTraceDepth).(*traceCtxInfo); ok {
		return rv
	}

	// Check metdata incoming for parent span and return it
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		value := md.Get("spanhelper-parent")
		if len(value) > 0 {
			if rv, ok := localTraceInfo.Load(value[len(value)-1]); ok {
				return rv.(*traceCtxInfo)
			}
		}
	}

	return nil
}

// LogFromSpan - return a logger that has a TraceHook to also log messages to the span
func LogFromSpan(span opentracing.Span) logrus.FieldLogger {
	if span == nil {
		return logrus.StandardLogger()
	}
	hookInit.Do(func() {
		logrus.AddHook(&traceHook{})
	})

	return logrus.StandardLogger().
		WithField(spanField, span).
		WithContext(context.WithValue(
			context.Background(),
			spanHelperState,
			&traceHookState{span: span},
		))
}

type traceHook struct{}

type traceHookState struct {
	index int32
	span  opentracing.Span
}

func (h *traceHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *traceHook) Fire(entry *logrus.Entry) error {
	if entry.Context == nil {
		return nil
	}
	state, ok := entry.Context.Value(spanHelperState).(*traceHookState)
	if !ok {
		return nil
	}

	idx := atomic.AddInt32(&state.index, 1) - 1
	fields := []log.Field{
		log.String(
			fmt.Sprintf("log[%v]", idx),
			fmt.Sprintf("[%s] %s",
				strings.ToUpper(entry.Level.String()),
				entry.Message),
		),
	}
	for key, val := range entry.Data {
		if key == spanField || key == operationField {
			continue
		}
		str, ok := val.(string)
		if !ok {
			str = fmt.Sprint(val)
		}
		fields = append(fields, log.String(key, str))
	}
	state.span.LogFields(fields...)

	return nil
}

// SpanHelper - wrap span if specified to simplify workflow
type SpanHelper interface {
	Finish()
	Context() context.Context
	Logger() logrus.FieldLogger
	LogObject(attribute string, value interface{})
	LogValue(attribute string, value interface{})
	LogError(err error)
	LogErrorf(format string, err error)
	Span() opentracing.Span
}

type spanHelper struct {
	operation string
	span      opentracing.Span
	ctx       context.Context
	logger    logrus.FieldLogger
	info      *traceCtxInfo
}

func (s *spanHelper) Span() opentracing.Span {
	return s.span
}

func (s *spanHelper) LogError(err error) {
	s.LogErrorf("%+v", err)
}

func (s *spanHelper) LogErrorf(format string, err error) {
	if s.span != nil && err != nil {
		d := limitString(string(debug.Stack()))
		msg := limitString(fmt.Sprintf(format, err))
		otgrpc.SetSpanTags(s.span, err, false)
		s.span.LogFields(log.String("event", "error"), log.String("message", msg), log.String("stacktrace", d))
		s.logEntry().Errorf("%v %s %s=%v%v", s.info.incInfo(), strings.Repeat(separator, s.info.level), "error", fmt.Sprintf(format, err), s.getSpan())
	}
}

func (s *spanHelper) logEntry() *logrus.Entry {
	e := toolsLog.Entry(s.ctx)
	if len(e.Data) > 0 {
		return logrus.WithFields(e.Data)
	}
	return e
}

func (s *spanHelper) getSpan() string {
	spanStr := fmt.Sprintf("%v", s.span)
	if len(spanStr) > 0 && spanStr != "{}" && s.span != nil {
		return fmt.Sprintf(" span=%v", spanStr)
	}
	return ""
}

func (s *spanHelper) LogObject(attribute string, value interface{}) {
	cc, err := json.Marshal(value)
	msg := ""
	if err == nil {
		msg = string(cc)
	} else {
		msg = fmt.Sprint(msg)
	}
	if s.span != nil {
		s.span.LogFields(log.Object(attribute, limitString(msg)))
	}
	s.logEntry().Infof("%v %s %s=%v%v", s.info.incInfo(), strings.Repeat(separator, s.info.level), attribute, msg, s.getSpan())
}

func (s *spanHelper) LogValue(attribute string, value interface{}) {
	if s.span != nil {
		s.span.LogFields(log.Object(attribute, limitString(fmt.Sprint(value))))
	}
	s.logEntry().Infof("%v %s %s=%v%v", s.info.incInfo(), strings.Repeat(separator, s.info.level), attribute, value, s.getSpan())
}

func (s *spanHelper) Finish() {
	if s.span != nil {
		s.span.Finish()
		s.span = nil
	}
	localTraceInfo.Delete(s.info.id)
}

func (s *spanHelper) Logger() logrus.FieldLogger {
	if s.logger == nil {
		s.logger = LogFromSpan(s.span)
		if s.operation != "" {
			s.logger = s.logger.WithField(operationField, s.operation)
		}
	}
	return s.logger
}

// NewSpanHelper - constructs a span helper from context/snap and operation name.
func NewSpanHelper(ctx context.Context, span opentracing.Span, operation string) SpanHelper {
	var info *traceCtxInfo
	ctx, info = withTraceInfo(ctx)

	localTraceInfo.Store(info.id, info)

	return &spanHelper{
		ctx:       ctx,
		span:      span,
		operation: operation,
		info:      info,
	}
}

func (s *spanHelper) Context() context.Context {
	return s.ctx
}

// FromContext - return span helper from context and if opentracing is enabled start new span
func FromContext(ctx context.Context, operation string) (result SpanHelper) {
	if jaeger.IsOpentracingEnabled() {
		newSpan, newCtx := opentracing.StartSpanFromContext(ctx, operation)
		result = NewSpanHelper(newCtx, newSpan, operation)
	} else {
		// return just context
		result = NewSpanHelper(ctx, nil, operation)
	}
	result.(*spanHelper).printStart(operation)
	return result
}

func (s *spanHelper) printStart(operation string) {
	prefix := strings.Repeat(startSeparator, s.info.level)
	s.logEntry().Infof("%v%s⎆ %v()%v", s.info.incInfo(), prefix, operation, s.getSpan())
}

// GetSpanHelper - construct a span helper object from current context span
func GetSpanHelper(ctx context.Context) SpanHelper {
	if jaeger.IsOpentracingEnabled() {
		span := opentracing.SpanFromContext(ctx)
		return NewSpanHelper(ctx, span, "")
	}
	// return just context
	return &spanHelper{
		span: nil,
		ctx:  ctx,
	}
}

// CopySpan - construct span helper object with ctx and copy span from spanContext
// Will start new operation on span
func CopySpan(ctx context.Context, spanContext SpanHelper, operation string) SpanHelper {
	return WithSpan(ctx, spanContext.Span(), operation)
}

// WithSpan - construct span helper object with ctx and copy spanid from span
// Will start new operation on span
func WithSpan(ctx context.Context, span opentracing.Span, operation string) (result SpanHelper) {
	if jaeger.IsOpentracingEnabled() && span != nil {
		ctx = opentracing.ContextWithSpan(ctx, span)
		newSpan, newCtx := opentracing.StartSpanFromContext(ctx, operation)
		result = NewSpanHelper(newCtx, newSpan, operation)
	} else {
		result = NewSpanHelper(ctx, nil, operation)
	}
	result.(*spanHelper).printStart(operation)
	return result
}

func limitString(s string) string {
	if len(s) > maxStringLength {
		return s[maxStringLength-dotCount:] + strings.Repeat(".", dotCount)
	}
	return s
}
