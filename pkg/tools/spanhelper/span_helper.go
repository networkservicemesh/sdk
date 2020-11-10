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
	maxStringLength      int               = 1000
	dotCount             int               = 3
	separator            string            = " "
	startSeparator       string            = " "
)

var localTraceInfo sync.Map

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
	logger := logrus.Logger{
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
	if span != nil {
		logger := logger.WithField("span", span)
		logger.Logger.AddHook(NewTraceHook(span))
		return logger
	}
	return &logger
}

type traceHook struct {
	index int
	span  opentracing.Span
}

// NewTraceHook - create a TraceHook for also logging to a span
func NewTraceHook(span opentracing.Span) logrus.Hook {
	return &traceHook{
		span: span,
	}
}

func (h *traceHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *traceHook) Fire(entry *logrus.Entry) error {
	msg, err := entry.String()
	if err != nil {
		return err
	}
	h.span.LogFields(log.String(fmt.Sprintf("log[%d]", h.index), msg))
	h.index++
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
			s.logger = s.logger.WithField("operation", s.operation)
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
