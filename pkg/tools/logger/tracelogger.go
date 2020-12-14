package logger

import (
	"context"
	"fmt"
	"strings"

	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

type traceLogger struct {
	operation string
	span      opentracing.Span
	ctx       context.Context
	info      *traceCtxInfo
	entry     *logrus.Entry
}

func (s *traceLogger) Info(v ...interface{}) {
	s.Log(INFO, v)
}

func (s *traceLogger) Infof(format string, v ...interface{}) {
	s.Logf(INFO, format, v...)
}

func (s *traceLogger) Warn(v ...interface{}) {
	s.Log(WARN, v)
}

func (s *traceLogger) Warnf(format string, v ...interface{}) {
	s.Logf(WARN, format, v...)
}

func (s *traceLogger) Error(v ...interface{}) {
	s.Log(ERROR, v)
}

func (s *traceLogger) Errorf(format string, v ...interface{}) {
	s.Logf(ERROR, format, v...)
}

func (s *traceLogger) Fatal(v ...interface{}) {
	s.Log(FATAL, v)
}

func (s *traceLogger) Fatalf(format string, v ...interface{}) {
	s.Logf(FATAL, format, v...)
}

func TraceLog_C(ctx context.Context) Logger {
	return TraceLog_CO(ctx, SPANLOGGER_OP_UNTITLED)
}

func TraceLog_CO(ctx context.Context, operation string) Logger {
	var span opentracing.Span
	if jaeger.IsOpentracingEnabled() {
		span, ctx = opentracing.StartSpanFromContext(ctx, operation)
	}
	return TraceLog_COS(ctx, operation, span)
}

func TraceLog_COS(ctx context.Context, operation string, span opentracing.Span) Logger {
	var info *traceCtxInfo
	ctx, info = withTraceInfo(ctx)
	localTraceInfo.Store(info.id, info)
	if value, ok := ctx.Value(CTXKEY_LOGRUS_ENTRY).(*logrus.Entry); ok {
		logger := &traceLogger{}
		ctx = context.WithValue(ctx, CTXKEY_LOGGER, logger)
		logger.entry = value.WithContext(ctx)
		logger.printStart(operation)
		return logger
	}
	return nil
}

func (s *traceLogger) WithField(key, value interface{}) Logger {
	entry := s.entry
	entry = entry.WithFields(logrus.Fields{key.(string): value})
	logger := &traceLogger{span: s.span, entry: entry}
	ctx := context.WithValue(entry.Context, CTXKEY_LOGRUS_ENTRY, entry)
	ctx = context.WithValue(ctx, CTXKEY_LOGGER, logger)
	entry.Context = ctx
	return logger
}

func (s *traceLogger) Log(level loggerLevel, v ...interface{}) {
	s.Logf(level, format(v), v)
}

func (s *traceLogger) Logf(level loggerLevel, format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	s.entry.Tracef("%v %s %s=%v%v", s.info.incInfo(), strings.Repeat(separator, s.info.level), levelName(level), msg, s.getSpan())
}

func (s *traceLogger) getSpan() string {
	spanStr := fmt.Sprintf("%v", s.span)
	if len(spanStr) > 0 && spanStr != "{}" && s.span != nil {
		return fmt.Sprintf(" span=%v", spanStr)
	}
	return ""
}

func (s *traceLogger) printStart(operation string) {
	prefix := strings.Repeat(startSeparator, s.info.level)
	s.entry.Tracef("%v%s⎆ %v()%v", s.info.incInfo(), prefix, operation, s.getSpan())
}
