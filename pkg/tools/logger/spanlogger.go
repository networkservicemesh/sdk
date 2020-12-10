package logger

import (
	"context"
	"fmt"
	"strings"

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

func (s *spanLogger) Trace(v ...interface{}) {
	s.Log(TRACE, v)
}

func (s *spanLogger) Tracef(format string, v ...interface{}) {
	s.Logf(TRACE, format, v...)
}

func (s *spanLogger) WithField(key, value interface{}) Logger {
	data := make(entriesType, len(s.entries)+1)
	for k, v := range s.entries {
		data[k] = v
	}
	/*
		isErrField := false
		if t := reflect.TypeOf(value); t != nil {
			switch t.Kind() {
			case reflect.Func:
				isErrField = true
			case reflect.Ptr:
				isErrField = t.Elem().Kind() == reflect.Func
			}
		}
		if !isErrField {
			data[key] = value
		} else {
			panic(fmt.Sprintf("can not add field %q", key))
		}
	*/
	data[key] = value
	return &spanLogger{
		ctx:       s.ctx,
		span:      s.span,
		operation: s.operation,
		level:     s.level,
		entries:   data,
	}
	//return SpanLog(s.ctx, s.span , s.operation, s.level, data)
	//level     LoggerLevels)
}

func Format(v ...interface{}) string {
	return strings.Trim(strings.Repeat("%+v"+separator, len(v)), separator)
}

func (s *spanLogger) Log(level loggerLevel, v ...interface{}) {
	format := Format(v)
	s.Logf(level, format, v)
}

func (s *spanLogger) Logf(level loggerLevel, format string, v ...interface{}) {
	if !s.IsLevelEnabled(level) {
		return
	}
	if s.span != nil {
		if v != nil {
			msg := limitString(fmt.Sprintf(format, v...))
			s.span.LogFields(log.String("event", LevelName(level)), log.String("message", msg))
			for k, v := range s.entries {
				s.span.LogKV(k, v)
			}

		} else {
			panic("values array is nil")
		}
	} else {
		panic("span is nil")
	}
	/*
		for k, v := range logfdata.payload {
			s.span.LogKV(k, limitString(v.(string)))
		}
	*/
	//s.logEntry().Tracef("%v %s %s=%v%v", s.info.incInfo(), strings.Repeat(separator, s.info.level), "error", fmt.Sprintf(format, v...), s.getSpan())
	//s.logEntry().((logTypesMap[event]).(func(format string, args... interface{})))("")
}

func (s *spanLogger) IsLevelEnabled(level loggerLevel) bool {
	return s.level >= level
}

func limitString(s string) string {
	if len(s) > maxStringLength {
		return s[maxStringLength-dotCount:] + strings.Repeat(".", dotCount)
	}
	return s
}

func SpanLog(ctx context.Context, span opentracing.Span, operation string, level loggerLevel, entries entriesType) Logger {
	//var info *traceCtxInfo
	//ctx, info = withTraceInfo(ctx)
	if span == nil {
		if jaeger.IsOpentracingEnabled() {
			span, ctx = opentracing.StartSpanFromContext(ctx, operation)
		}
	}
	if entries == nil {
		entries = make(entriesType)
	}
	logger := &spanLogger{
		span:      span,
		operation: operation,
		//info:      info,
		level:   level,
		entries: entries,
	}
	logger.ctx = context.WithValue(ctx, CTXKEY_LOGGER, logger)
	/*
		if jaeger.IsOpentracingEnabled() {
			newSpan, newCtx := opentracing.StartSpanFromContext(ctx, operation)
			result = NewSpanHelper(newCtx, newSpan, operation)
		} else {
			// return just context
			result = NewSpanHelper(ctx, nil, operation)
		}
		result.(*spanHelper).printStart(operation)
		return result
	*/
	//localTraceInfo.Store(info.id, info)
	return logger
}

func (s *spanLogger) Close() {
	if s.span != nil {
		s.span.Finish()
		s.span = nil
	}
	//localTraceInfo.Delete(s.info.id)
}

/*
func (s *spanLogger) logEntry() *logrus.Entry {
	e := toolsLog.Entry(s.ctx)
	if len(e.Data) > 0 {
		return logrus.WithFields(e.Data)
	}
	return e
}

func (s *spanLogger) getSpan() string {
	spanStr := fmt.Sprintf("%v", s.span)
	if len(spanStr) > 0 && spanStr != "{}" && s.span != nil {
		return fmt.Sprintf(" span=%v", spanStr)
	}
	return ""
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

func (i *traceCtxInfo) incInfo() string {
	i.childCount++
	if i.childCount > 1 {
		return fmt.Sprintf("(%d.%d)", i.level, i.childCount-1)
	}
	return fmt.Sprintf("(%d)", i.level)
}


//used in withTraceInfo
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
*/
