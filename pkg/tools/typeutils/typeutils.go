// Copyright (c) 2020-2024 Cisco Systems, Inc.
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
	"strings"
)

// GetFuncName - returns the function name from the passed value (interface) and method name
func GetFuncName(value interface{}, methodName string) string {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	typeName := v.Type().Name()
	pkgPath := v.Type().PkgPath()
	pkgPath = pkgPath[strings.LastIndex(pkgPath, "/")+1:]
	sb := strings.Builder{}

	sb.Grow(len(methodName) + len(pkgPath) + len(typeName) + 2)
	_, _ = sb.WriteString(pkgPath)
	_, _ = sb.WriteString("/")
	_, _ = sb.WriteString(typeName)
	_, _ = sb.WriteString(".")
	_, _ = sb.WriteString(methodName)

	return sb.String()
}
