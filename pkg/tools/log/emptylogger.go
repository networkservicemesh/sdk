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

type emptylogger struct{}

func (s *emptylogger) Info(v ...interface{})                   {}
func (s *emptylogger) Infof(format string, v ...interface{})   {}
func (s *emptylogger) Warn(v ...interface{})                   {}
func (s *emptylogger) Warnf(format string, v ...interface{})   {}
func (s *emptylogger) Error(v ...interface{})                  {}
func (s *emptylogger) Errorf(format string, v ...interface{})  {}
func (s *emptylogger) Fatal(v ...interface{})                  { os.Exit(1) }
func (s *emptylogger) Fatalf(format string, v ...interface{})  { os.Exit(1) }
func (s *emptylogger) Debug(v ...interface{})                  {}
func (s *emptylogger) Debugf(format string, v ...interface{})  {}
func (s *emptylogger) Trace(v ...interface{})                  {}
func (s *emptylogger) Tracef(format string, v ...interface{})  {}
func (s *emptylogger) Object(k, v interface{})                 {}
func (s *emptylogger) WithField(key, value interface{}) Logger { return s }

// Empty - provides a logger that does nothing
func Empty() Logger {
	return &emptylogger{}
}
