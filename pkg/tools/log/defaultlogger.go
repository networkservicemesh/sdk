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

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

type defaultLogger struct {
	prefix string
}

// Default - provides a default logger
func Default() Logger {
	return &defaultLogger{}
}

func (l *defaultLogger) Info(v ...interface{}) {
	log.Println(l.msg("[INFO] ", v...))
}

func (l *defaultLogger) Infof(format string, v ...interface{}) {
	l.Info(fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Warn(v ...interface{}) {
	log.Println(l.msg("[WARN] ", v...))
}

func (l *defaultLogger) Warnf(format string, v ...interface{}) {
	l.Warn(fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Error(v ...interface{}) {
	log.Println(l.msg("[ERROR]", v...))
}

func (l *defaultLogger) Errorf(format string, v ...interface{}) {
	l.Error(fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Fatal(v ...interface{}) {
	log.Fatalln(l.msg("[FATAL]", v...))
}

func (l *defaultLogger) Fatalf(format string, v ...interface{}) {
	l.Fatal(fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Debug(v ...interface{}) {
	log.Println(l.msg("[DEBUG]", v...))
}

func (l *defaultLogger) Debugf(format string, v ...interface{}) {
	l.Debug(fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Trace(v ...interface{}) {
	log.Println(l.msg("[TRACE]", v...))
}

func (l *defaultLogger) Tracef(format string, v ...interface{}) {
	l.Trace(fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Object(k, v interface{}) {
	msg := ""
	cc, err := json.Marshal(v)
	if err == nil {
		msg = string(cc)
	} else {
		msg = fmt.Sprint(v)
	}
	l.Infof("%v=%s", k, msg)
}

func (l *defaultLogger) WithField(key, value interface{}) Logger {
	var prefix string
	if l.prefix != "" {
		prefix = fmt.Sprintf("%s [%s:%s]", l.prefix, key, value)
	} else {
		prefix = fmt.Sprintf("[%s:%s]", key, value)
	}

	return &defaultLogger{
		prefix: prefix,
	}
}

func (l *defaultLogger) msg(level string, v ...interface{}) string {
	sb := strings.Builder{}

	sb.WriteString(level)
	sb.WriteRune(' ')

	if l.prefix != "" {
		sb.WriteString(l.prefix)
		sb.WriteRune(' ')
	}

	sb.WriteString(fmt.Sprint(v...))

	return sb.String()
}
