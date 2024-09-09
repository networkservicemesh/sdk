// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Field structure that is used for additional information in the logs.
type Field struct {
	k string
	v interface{}
}

// NewField creates Field.
func NewField(k string, v interface{}) *Field {
	return &Field{k, v}
}

// Key returns key.
func (f *Field) Key() string {
	return f.k
}

// Val returns value.
func (f *Field) Val() interface{} {
	return f.v
}
