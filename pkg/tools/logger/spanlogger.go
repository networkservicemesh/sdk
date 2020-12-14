package logger

import (
	"context"
	"fmt"

	//toolsLog "github.com/networkservicemesh/sdk/pkg/tools/log"
	//"github.com/sirupsen/logrus"
	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

//var localTraceInfo sync.Map

/*
type traceCtxInfo struct {
	level      int
	childCount int
	id         string
}
*/

type spanLogger struct {
	operation string
	span      opentracing.Span
	ctx       context.Context
	//info      *traceCtxInfo
	//entryPool sync.Pool
	entries entriesType
	level   loggerLevel
}

func (s *spanLogger) Info(v ...interface{}) {
	s.Log(INFO, v)
}

func (s *spanLogger) Infof(format string, v ...interface{}) {
	s.Logf(INFO, format, v...)
}

func (s *spanLogger) Warn(v ...interface{}) {
	s.Log(WARN, v)
}

func (s *spanLogger) Warnf(format string, v ...interface{}) {
	s.Logf(WARN, format, v...)
}

func (s *spanLogger) Error(v ...interface{}) {
	s.Log(ERROR, v)
}

func (s *spanLogger) Errorf(format string, v ...interface{}) {
	s.Logf(ERROR, format, v...)
}

func (s *spanLogger) Fatal(v ...interface{}) {
	s.Log(FATAL, v)
}

func (s *spanLogger) Fatalf(format string, v ...interface{}) {
	s.Logf(FATAL, format, v...)
}

func (s *spanLogger) WithField(key, value interface{}) Logger {
	data := make(entriesType, len(s.entries)+1)
	for k, v := range s.entries {
		data[k] = v
	}
	data[key] = value
	logger := &spanLogger{
		ctx:       s.ctx,
		span:      s.span,
		operation: s.operation,
		level:     s.level,
		entries:   data,
	}
	logger.ctx = context.WithValue(s.ctx, CTXKEY_LOGGER, logger)
	return logger
}

func (s *spanLogger) Log(level loggerLevel, v ...interface{}) {
	s.Logf(level, format(v), v)
}

func (s *spanLogger) Logf(level loggerLevel, format string, v ...interface{}) {
	if s.span != nil {
		if v != nil {
			msg := limitString(fmt.Sprintf(format, v...))
			s.span.LogFields(log.String("event", levelName(level)), log.String("message", msg))
			for k, v := range s.entries {
				s.span.LogKV(k, v)
			}
		} else {
			panic("values array is nil")
		}
	} else {
		panic("span is nil")
	}
}

func SpanLog_C(ctx context.Context) Logger {
	return SpanLog_CO(ctx, SPANLOGGER_OP_UNTITLED)
}

func SpanLog_CO(ctx context.Context, operation string) Logger {
	var span opentracing.Span
	if jaeger.IsOpentracingEnabled() {
		span, ctx = opentracing.StartSpanFromContext(ctx, operation)
	}
	return SpanLog_COS(ctx, operation, span)
}

func SpanLog_COS(ctx context.Context, operation string, span opentracing.Span) Logger {
	logger := &spanLogger{
		span:      span,
		operation: operation,
		entries:   make(entriesType),
	}
	logger.ctx = context.WithValue(ctx, CTXKEY_LOGGER, logger)
	return logger
}

func (s *spanLogger) Close() {
	if s.span != nil {
		s.span.Finish()
		s.span = nil
	}
}
