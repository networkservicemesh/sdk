package logger

import (
	"context"
)

type loggerLevel int
type entriesType map[interface{}]interface{}
type contextKeyType string
type loggerType string
type traceLoggerKeyType string

const (
	INFO loggerLevel = iota
	WARN
	ERROR
	FATAL
	logrusEntry            contextKeyType     = "LogrusEntry"
	CTXKEY_LOGGER          contextKeyType     = "logger"
	CTXKEY_LOGRUS_ENTRY    contextKeyType     = "logrusEntry"
	SPANLOGGER_OP_UNTITLED string             = "Untitled operation"
	traceInfoKey           contextKeyType     = "ConnectionInfo"
	traceLoggerTraceDepth  traceLoggerKeyType = "traceLoggerTraceDepth"
	maxStringLength        int                = 1000
	dotCount               int                = 3
	separator              string             = " "
	startSeparator         string             = " "
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

	WithField(key, value interface{}) Logger
}

func Log(ctx context.Context) Logger {
	if ctx != nil {
		if value := ctx.Value(CTXKEY_LOGGER); value != nil {
			return value.(Logger)
		}
	}
	return nil
}
