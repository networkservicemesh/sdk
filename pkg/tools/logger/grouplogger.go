package logger

import "context"

type groupLogger struct {
	loggers []Logger
}

/*
func GroupLogFromTypes(ctx context.Context, types []loggerType) Logger {
	loggers := make([]Logger, len(types))
	for i, t := range types {
		if loggers[i] = CreateLogger_CT(ctx, t); loggers[i] == nil {
			return nil
		}
	}
	return &groupLogger{loggers: loggers}
}
*/

func GroupLogFromSlice(ctx context.Context, s []Logger) Logger {
	loggers := make([]Logger, len(s))
	for i, t := range s {
		if loggers[i] = t; loggers[i] == nil {
			return nil
		}
	}
	return &groupLogger{loggers: loggers}
}

func (j *groupLogger) Info(v ...interface{}) {
	for _, l := range j.loggers {
		l.Info(v...)
	}
}

func (j *groupLogger) Infof(format string, v ...interface{}) {
	for _, l := range j.loggers {
		l.Infof(format, v...)
	}
}

func (j *groupLogger) Warn(v ...interface{}) {
	for _, l := range j.loggers {
		l.Warn(v...)
	}
}

func (j *groupLogger) Warnf(format string, v ...interface{}) {
	for _, l := range j.loggers {
		l.Warnf(format, v...)
	}
}

func (j *groupLogger) Error(v ...interface{}) {
	for _, l := range j.loggers {
		l.Error(v...)
	}
}

func (j *groupLogger) Errorf(format string, v ...interface{}) {
	for _, l := range j.loggers {
		l.Errorf(format, v...)
	}
}

func (j *groupLogger) Fatal(v ...interface{}) {
	for _, l := range j.loggers {
		l.Fatal(v...)
	}
}

func (j *groupLogger) Fatalf(format string, v ...interface{}) {
	for _, l := range j.loggers {
		l.Fatalf(format, v...)
	}
}

func (j *groupLogger) Trace(v ...interface{}) {
	for _, l := range j.loggers {
		l.Trace(v...)
	}
}

func (j *groupLogger) Tracef(format string, v ...interface{}) {
	for _, l := range j.loggers {
		l.Tracef(format, v...)
	}
}

func (j *groupLogger) WithField(key, value interface{}) Logger {
	loggers := make([]Logger, len(j.loggers))
	for i, l := range j.loggers {
		loggers[i] = l
	}
	return &groupLogger{loggers: loggers}
}

//var _ Logger = (*groupLogger(nil))
