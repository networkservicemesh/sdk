// Copyright (c) 2020 Cisco Systems, Inc.
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

// Package typeutils provides as simpler helper for getting a type Name from an interface{}
package typeutils

import (
	"reflect"
	"runtime"
)

// GetFuncName - returns the function name from the passed function pointer
func GetFuncName(value interface{}) string {
	t := reflect.TypeOf(value)
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(value).Pointer()).Name()
	}
	return ""
}
