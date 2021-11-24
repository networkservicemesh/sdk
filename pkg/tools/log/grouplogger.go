// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package log

import "os"

// groupLogger - combines multiple loggers
type groupLogger struct {
	loggers []Logger
}

// Combine creates grouped logger from all provided loggers
func Combine(s ...Logger) Logger {
	return &groupLogger{loggers: s}
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
	os.Exit(1)
}

func (j *groupLogger) Fatalf(format string, v ...interface{}) {
	for _, l := range j.loggers {
		l.Fatalf(format, v...)
	}
	os.Exit(1)
}

func (j *groupLogger) Debug(v ...interface{}) {
	for _, l := range j.loggers {
		l.Debug(v...)
	}
}

func (j *groupLogger) Debugf(format string, v ...interface{}) {
	for _, l := range j.loggers {
		l.Debugf(format, v...)
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

func (j *groupLogger) Object(k, v interface{}) {
	for _, l := range j.loggers {
		l.Object(k, v)
	}
}

func (j *groupLogger) WithField(key, value interface{}) Logger {
	loggers := make([]Logger, len(j.loggers))
	for i, l := range j.loggers {
		loggers[i] = l.WithField(key, value)
	}
	return &groupLogger{loggers: loggers}
}
