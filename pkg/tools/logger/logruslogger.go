package logger

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

/*
// WithFields - return new context with fields added to Context's logrus.Entry
func WithFields(parent context.Context, fields logrus.Fields) context.Context {
	entry := Entry(parent, entry.level)
	entry = entry.WithFields(fields)
	ctx := context.WithValue(parent, logrusEntry, entry)
	entry.Context = ctx
	return ctx
}

// WithField - return new context with {key:value} added to Context's logrus.Entry
func WithField(parent context.Context, key string, value interface{}) context.Context {
	return WithFields(parent, logrus.Fields{
		key: value,
	})
}
*/

// Entry - returns *logrus.Entry for context.  Note: each context has its *own* Entry with Entry.Context set
//         to that context (so context values can be used in logrus.Hooks)

func LogrusLevel(level loggerLevel) logrus.Level {
	switch level {
	case INFO:
		return logrus.InfoLevel
	case WARN:
		return logrus.WarnLevel
	case ERROR:
		return logrus.ErrorLevel
	case FATAL:
		return logrus.FatalLevel
	case TRACE:
		return logrus.TraceLevel
	}
	return logrus.DebugLevel
}

func Entry(ctx context.Context, level loggerLevel) *logrus.Entry {
	/*
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
	*/
	entry := logrus.WithTime(time.Now())
	entry.Level = LogrusLevel(level)
	return entry
}

func LogrusLog(ctx context.Context, level loggerLevel) Logger {
	/*
		logger := &logrus.Logger{
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
	*/
	logger := &regularLogger{}
	ctx = context.WithValue(ctx, CTXKEY_LOGGER, logger)
	logger.entry = Entry(ctx, level)
	return logger
}

type regularLogger struct {
	entry *logrus.Entry
}

func (r *regularLogger) Info(v ...interface{}) {
	r.entry.Info(v...)
}

func (r *regularLogger) Infof(format string, v ...interface{}) {
	r.entry.Infof(format, v...)
}

func (r *regularLogger) Warn(v ...interface{}) {
	r.entry.Warn(v...)
}

func (r *regularLogger) Warnf(format string, v ...interface{}) {
	r.entry.Warnf(format, v...)
}

func (r *regularLogger) Error(v ...interface{}) {
	r.entry.Error(v...)
}

func (r *regularLogger) Errorf(format string, v ...interface{}) {
	r.entry.Errorf(format, v...)
}

func (r *regularLogger) Fatal(v ...interface{}) {
	r.entry.Error(v...)
}

func (r *regularLogger) Fatalf(format string, v ...interface{}) {
	r.entry.Errorf(format, v...)
}

func (r *regularLogger) Trace(v ...interface{}) {
	r.entry.Error(v...)
}

func (r *regularLogger) Tracef(format string, v ...interface{}) {
	r.entry.Errorf(format, v...)
}

func (r *regularLogger) WithField(key, value interface{}) Logger {
	entry := r.entry
	entry = entry.WithFields(logrus.Fields{key.(string): value})
	ctx := context.WithValue(r.entry.Context, logrusEntry, entry)
	entry.Context = ctx
	return &regularLogger{entry: entry}
}
