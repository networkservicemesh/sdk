package logger

import (
	"context"

	"github.com/sirupsen/logrus"
)

func LogrusLog(ctx context.Context) Logger {
	if value, ok := ctx.Value(CTXKEY_LOGRUS_ENTRY).(*logrus.Entry); ok {
		logger := &logrusLogger{}
		ctx = context.WithValue(ctx, CTXKEY_LOGGER, logger)
		logger.entry = value.WithContext(ctx)
		return logger
	}
	return nil
}

type logrusLogger struct {
	entry *logrus.Entry
}

func (r *logrusLogger) Info(v ...interface{}) {
	r.entry.Info(v...)
}

func (r *logrusLogger) Infof(format string, v ...interface{}) {
	r.entry.Infof(format, v...)
}

func (r *logrusLogger) Warn(v ...interface{}) {
	r.entry.Warn(v...)
}

func (r *logrusLogger) Warnf(format string, v ...interface{}) {
	r.entry.Warnf(format, v...)
}

func (r *logrusLogger) Error(v ...interface{}) {
	r.entry.Error(v...)
}

func (r *logrusLogger) Errorf(format string, v ...interface{}) {
	r.entry.Errorf(format, v...)
}

func (r *logrusLogger) Fatal(v ...interface{}) {
	r.entry.Error(v...)
}

func (r *logrusLogger) Fatalf(format string, v ...interface{}) {
	r.entry.Errorf(format, v...)
}

func (r *logrusLogger) Trace(v ...interface{}) {
	r.entry.Error(v...)
}

func (r *logrusLogger) Tracef(format string, v ...interface{}) {
	r.entry.Errorf(format, v...)
}

func (r *logrusLogger) WithField(key, value interface{}) Logger {
	entry := r.entry
	entry = entry.WithFields(logrus.Fields{key.(string): value})
	logger := &logrusLogger{entry: entry}
	ctx := context.WithValue(entry.Context, CTXKEY_LOGRUS_ENTRY, entry)
	ctx = context.WithValue(ctx, CTXKEY_LOGGER, logger)
	entry.Context = ctx
	return logger
}
