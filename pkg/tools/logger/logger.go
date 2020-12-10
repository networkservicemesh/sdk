package logger

import (
	"context"
)

type loggerLevel int
type entriesType map[interface{}]interface{}
type contextKeyType string
type loggerType string
type spanHelperKeyType string

const (
	TRACE                  loggerLevel       = 0
	INFO                   loggerLevel       = 100
	WARN                   loggerLevel       = 200
	ERROR                  loggerLevel       = 300
	FATAL                  loggerLevel       = 400
	logrusEntry            contextKeyType    = "LogrusEntry"
	CTXKEY_LOGGER          contextKeyType    = "logger"
	CTXKEY_LOGGERTYPE      contextKeyType    = "logger_type"
	CTXKEY_LOGGERLEVEL     contextKeyType    = "logger_level"
	CTXKEY_SPANLOGGER_OP   contextKeyType    = "spanlogger_operation"
	SPANLOGGER_OP_UNTITLED string            = "Untitled operation"
	LOGGERTYPE_LOGRUS      loggerType        = "logrus"
	LOGGERTYPE_SPAN        loggerType        = "span"
	LOGGERTYPE_GROUP       loggerType        = "group"
	CTXKEY_GL_TYPES        contextKeyType    = "grouplogger_types"
	CTXKEY_GL_SLICE        contextKeyType    = "grouplogger_slice"
	traceInfoKey           contextKeyType    = "ConnectionInfo"
	spanHelperTraceDepth   spanHelperKeyType = "spanHelperTraceDepth"
	maxStringLength        int               = 1000
	dotCount               int               = 3
	separator              string            = " "
	startSeparator         string            = " "
)

type Logger interface {
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Warn(v ...interface{})
	Warnf(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Trace(v ...interface{})
	Tracef(format string, v ...interface{})

	WithField(key, value interface{}) Logger
}

func CreateLogger_CTL(ctx context.Context, logger_type loggerType, level loggerLevel) Logger {
	if logger_type == LOGGERTYPE_SPAN {
		if value, ok := ctx.Value(CTXKEY_SPANLOGGER_OP).(string); ok {
			return SpanLog(ctx, nil, value, level, nil)
		}
		return SpanLog(ctx, nil, SPANLOGGER_OP_UNTITLED, TRACE, nil)
	} else if logger_type == LOGGERTYPE_LOGRUS {
		return LogrusLog(ctx, level)
	}
	return nil
}

func CreateLogger_CT(ctx context.Context, logger_type loggerType) Logger {
	if logger_type == LOGGERTYPE_GROUP {
		if value, ok := ctx.Value(CTXKEY_GL_SLICE).([]Logger); ok {
			return GroupLogFromSlice(ctx, value)
		} /* else if value, ok := ctx.Value(CTXKEY_GL_SLICE).([]loggerType); ok {
			GroupLogFromTypes(ctx, value)
		}*/
		return nil
	}
	if value, ok := ctx.Value(CTXKEY_LOGGERLEVEL).(loggerLevel); ok {
		return CreateLogger_CTL(ctx, logger_type, value)
	}
	return nil
}

func Log(ctx context.Context) Logger {
	if ctx != nil {
		if value := ctx.Value(CTXKEY_LOGGER); value != nil {
			return value.(Logger)
		}
		if value, ok := ctx.Value(CTXKEY_LOGGERTYPE).(loggerType); ok {
			return CreateLogger_CT(ctx, value)
		}
	}
	return nil
}

func LevelName(level loggerLevel) string {
	levelNames := map[loggerLevel]string{
		INFO:  "info",
		WARN:  "warn",
		ERROR: "error",
		FATAL: "fatal",
		TRACE: "trace",
	}
	return levelNames[level]
}

/*
func Log(ctx context.Context) Logger {
	if ctx == nil {
		panic("cannot create context from nil parent")
	}
	if entryValue := ctx.Value(logrusEntry); entryValue != nil {
		if entry := entryValue.(*logrus.Entry); entry != nil {
			if entry.Context == ctx {
				return entry
			}
			return entry.WithContext(ctx)
		}
	}
	return logrus.WithTime(time.Now())
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
*/
