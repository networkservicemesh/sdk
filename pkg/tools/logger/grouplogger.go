package logger

type groupLogger struct {
	loggers []Logger
}

func GroupLogFromSlice(s []Logger) Logger {
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

func (j *groupLogger) WithField(key, value interface{}) Logger {

	loggers := make([]Logger, len(j.loggers))
	for i, l := range j.loggers {
		loggers[i] = l.WithField(key, value)
	}
	return &groupLogger{loggers: loggers}
}
